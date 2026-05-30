resource "google_project_service" "sdz_services" {
  for_each = toset([
    "cloudresourcemanager.googleapis.com",
    "firebase.googleapis.com",
    "firestore.googleapis.com",
    "iam.googleapis.com",
    "identitytoolkit.googleapis.com",
    "artifactregistry.googleapis.com",
    "iamcredentials.googleapis.com",
    "run.googleapis.com",
    "storage.googleapis.com",
    "sts.googleapis.com",
    "cloudbuild.googleapis.com",
  ])

  service            = each.value
  disable_on_destroy = false
}

data "google_project" "sdz_project" {
  project_id = var.sdz_project_id
}

resource "google_firebase_project" "sdz_firebase" {
  provider = google-beta
  project  = var.sdz_project_id

  depends_on = [google_project_service.sdz_services]
}

resource "google_firestore_database" "sdz_firestore" {
  provider    = google-beta
  project     = var.sdz_project_id
  name        = "(default)"
  location_id = var.sdz_firestore_location
  type        = "FIRESTORE_NATIVE"

  depends_on = [google_firebase_project.sdz_firebase]
}

#tfsec:ignore:google-storage-bucket-encryption-customer-key Public assets bucket; CMEK not required for this use case.
resource "google_storage_bucket" "sdz_spot_media" {
  name     = var.sdz_storage_bucket
  location = var.sdz_region

  uniform_bucket_level_access = true
  public_access_prevention    = "inherited"

  depends_on = [google_project_service.sdz_services]
}

#tfsec:ignore:google-storage-bucket-encryption-customer-key Static hosting bucket; CMEK not required for this use case.
resource "google_storage_bucket" "sdz_ui_bucket" {
  name     = var.sdz_ui_bucket
  location = var.sdz_region

  uniform_bucket_level_access = true
  public_access_prevention    = "inherited"

  website {
    main_page_suffix = "index.html"
    not_found_page   = "index.html"
  }

  depends_on = [google_project_service.sdz_services]
}

resource "google_storage_bucket_iam_member" "sdz_ui_public_read" {
  for_each = toset(var.sdz_ui_public_members)
  bucket   = google_storage_bucket.sdz_ui_bucket.name
  role     = "roles/storage.objectViewer"
  member   = each.value
}

resource "google_artifact_registry_repository" "sdz_api_repo" {
  location      = var.sdz_region
  repository_id = "sdz-${var.sdz_stage}-arapi"
  format        = "DOCKER"

  depends_on = [google_project_service.sdz_services]
}

resource "google_service_account" "sdz_dev_api_sa" {
  account_id   = "sdz-dev-saapi"
  display_name = "spot-diggz API用サービスアカウント"
  project      = var.sdz_project_id
}

resource "google_service_account" "sdz_dev_deploy_sa" {
  account_id   = "sdz-dev-sadeploy"
  display_name = "spot-diggz デプロイ用サービスアカウント"
  project      = var.sdz_project_id
}

resource "google_service_account" "sdz_dev_terraform_sa" {
  account_id   = "sdz-dev-satf"
  display_name = "spot-diggz terraform用サービスアカウント"
  project      = var.sdz_project_id
}

resource "google_iam_workload_identity_pool" "sdz_github" {
  count                     = var.sdz_enable_wif ? 1 : 0
  project                   = var.sdz_project_id
  workload_identity_pool_id = var.sdz_wif_pool_id
  display_name              = "sdz-github-pool"
  description               = "GitHub Actions OIDC pool"
}

resource "google_iam_workload_identity_pool_provider" "sdz_github" {
  count                              = var.sdz_enable_wif ? 1 : 0
  project                            = var.sdz_project_id
  workload_identity_pool_id          = var.sdz_wif_pool_id
  workload_identity_pool_provider_id = var.sdz_wif_provider_id
  display_name                       = "sdz-github-provider"
  description                        = "GitHub Actions OIDC provider"

  oidc {
    issuer_uri = "https://token.actions.githubusercontent.com"
  }

  attribute_mapping = {
    "google.subject"       = "assertion.sub"
    "attribute.repository" = "assertion.repository"
    "attribute.ref"        = "assertion.ref"
    "attribute.actor"      = "assertion.actor"
  }

  attribute_condition = var.sdz_github_ref != "" ? "attribute.repository == \"${var.sdz_github_repository}\" && attribute.ref == \"${var.sdz_github_ref}\"" : "attribute.repository == \"${var.sdz_github_repository}\""
}

resource "google_service_account" "sdz_firebase_adminsdk_sa" {
  account_id   = "firebase-adminsdk-fbsvc"
  display_name = "Firebase Admin SDK Service Agent"
  project      = var.sdz_project_id
}

resource "google_service_account_iam_member" "sdz_api_sa_token_creator" {
  service_account_id = google_service_account.sdz_dev_api_sa.name
  role               = "roles/iam.serviceAccountTokenCreator"
  member             = "serviceAccount:${google_service_account.sdz_dev_api_sa.email}"
}

resource "google_service_account_iam_member" "sdz_deploy_sa_cloud_build_user" {
  service_account_id = google_service_account.sdz_dev_deploy_sa.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${data.google_project.sdz_project.number}@cloudbuild.gserviceaccount.com"
}

resource "google_service_account_iam_member" "sdz_deploy_sa_self_user" {
  service_account_id = google_service_account.sdz_dev_deploy_sa.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.sdz_dev_deploy_sa.email}"
}

resource "google_service_account_iam_member" "sdz_api_sa_deploy_user" {
  service_account_id = google_service_account.sdz_dev_api_sa.name
  role               = "roles/iam.serviceAccountUser"
  member             = "serviceAccount:${google_service_account.sdz_dev_deploy_sa.email}"
}

resource "google_project_iam_member" "sdz_deploy_run_admin" {
  project = var.sdz_project_id
  role    = "roles/run.admin"
  member  = "serviceAccount:${google_service_account.sdz_dev_deploy_sa.email}"
}

resource "google_project_iam_member" "sdz_deploy_artifact_writer" {
  project = var.sdz_project_id
  role    = "roles/artifactregistry.writer"
  member  = "serviceAccount:${google_service_account.sdz_dev_deploy_sa.email}"
}

resource "google_project_iam_member" "sdz_deploy_cloudbuild_submitter" {
  project = var.sdz_project_id
  role    = "roles/cloudbuild.builds.editor"
  member  = "serviceAccount:${google_service_account.sdz_dev_deploy_sa.email}"
}

resource "google_project_iam_member" "sdz_api_firestore_user" {
  project = var.sdz_project_id
  role    = "roles/datastore.user"
  member  = "serviceAccount:${google_service_account.sdz_dev_api_sa.email}"
}

resource "google_service_account_iam_member" "sdz_deploy_sa_wif" {
  count              = var.sdz_enable_wif ? 1 : 0
  service_account_id = google_service_account.sdz_dev_deploy_sa.name
  role               = "roles/iam.workloadIdentityUser"
  member             = "principalSet://iam.googleapis.com/projects/${data.google_project.sdz_project.number}/locations/global/workloadIdentityPools/${var.sdz_wif_pool_id}/attribute.repository/${var.sdz_github_repository}"

  depends_on = [google_iam_workload_identity_pool_provider.sdz_github]
}

resource "google_storage_bucket_iam_member" "sdz_api_storage_admin" {
  bucket = google_storage_bucket.sdz_spot_media.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.sdz_dev_api_sa.email}"
}

resource "google_storage_bucket_iam_member" "sdz_deploy_ui_storage_admin" {
  bucket = google_storage_bucket.sdz_ui_bucket.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.sdz_dev_deploy_sa.email}"
}

resource "google_storage_bucket_iam_member" "sdz_deploy_cloudbuild_source_reader" {
  bucket = var.sdz_cloudbuild_source_bucket
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.sdz_dev_deploy_sa.email}"
}

resource "google_storage_bucket_iam_member" "sdz_deploy_cloudbuild_bucket_reader" {
  bucket = var.sdz_cloudbuild_source_bucket
  role   = "roles/storage.legacyBucketReader"
  member = "serviceAccount:${google_service_account.sdz_dev_deploy_sa.email}"
}

resource "google_storage_bucket_iam_member" "sdz_deploy_cloudbuild_bucket_writer" {
  bucket = var.sdz_cloudbuild_source_bucket
  role   = "roles/storage.legacyBucketWriter"
  member = "serviceAccount:${google_service_account.sdz_dev_deploy_sa.email}"
}

resource "google_project_iam_member" "sdz_deploy_service_usage_consumer" {
  project = var.sdz_project_id
  role    = "roles/serviceusage.serviceUsageConsumer"
  member  = "serviceAccount:${google_service_account.sdz_dev_deploy_sa.email}"
}

resource "google_project_iam_member" "sdz_deploy_log_writer" {
  project = var.sdz_project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.sdz_dev_deploy_sa.email}"
}

resource "google_storage_bucket_iam_member" "sdz_cloudbuild_sa_source_admin" {
  bucket = var.sdz_cloudbuild_source_bucket
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${data.google_project.sdz_project.number}@cloudbuild.gserviceaccount.com"
}

resource "google_storage_bucket_iam_member" "sdz_cloudbuild_sa_bucket_reader" {
  bucket = var.sdz_cloudbuild_source_bucket
  role   = "roles/storage.legacyBucketReader"
  member = "serviceAccount:${data.google_project.sdz_project.number}@cloudbuild.gserviceaccount.com"
}

resource "google_storage_bucket_iam_member" "sdz_cloudbuild_sa_bucket_writer" {
  bucket = var.sdz_cloudbuild_source_bucket
  role   = "roles/storage.legacyBucketWriter"
  member = "serviceAccount:${data.google_project.sdz_project.number}@cloudbuild.gserviceaccount.com"
}

resource "google_project_iam_member" "sdz_cloudbuild_sa_service_usage_consumer" {
  project = var.sdz_project_id
  role    = "roles/serviceusage.serviceUsageConsumer"
  member  = "serviceAccount:${data.google_project.sdz_project.number}@cloudbuild.gserviceaccount.com"
}

resource "google_storage_bucket_iam_member" "sdz_spot_media_public_read" {
  for_each = toset(var.sdz_media_public_members)
  bucket   = google_storage_bucket.sdz_spot_media.name
  role     = "roles/storage.objectViewer"
  member   = each.value
}

resource "google_cloud_run_service" "sdz_api" {
  count    = var.sdz_enable_cloud_run ? 1 : 0
  name     = "sdz-${var.sdz_stage}-api"
  location = var.sdz_region
  project  = var.sdz_project_id

  template {
    spec {
      service_account_name = google_service_account.sdz_dev_api_sa.email
      containers {
        image = var.sdz_api_image
        env {
          name  = "SDZ_AUTH_PROJECT_ID"
          value = var.sdz_auth_project_id
        }
        env {
          name  = "SDZ_USE_FIRESTORE"
          value = var.sdz_use_firestore ? "1" : "0"
        }
        env {
          name  = "SDZ_FIRESTORE_PROJECT_ID"
          value = var.sdz_firestore_project_id
        }
        env {
          name  = "SDZ_CORS_ALLOWED_ORIGINS"
          value = var.sdz_cors_allowed_origins
        }
        env {
          name  = "SDZ_STORAGE_BUCKET"
          value = var.sdz_storage_bucket
        }
        env {
          name  = "SDZ_STORAGE_SERVICE_ACCOUNT_EMAIL"
          value = var.sdz_storage_service_account_email
        }
        env {
          name  = "SDZ_STORAGE_SIGNED_URL_EXPIRES_SECS"
          value = tostring(var.sdz_storage_signed_url_expires_secs)
        }
      }
    }
  }

  traffic {
    percent         = 100
    latest_revision = true
  }

  depends_on = [google_project_service.sdz_services]
}

resource "google_cloud_run_service_iam_member" "sdz_api_public" {
  count    = var.sdz_enable_cloud_run ? 1 : 0
  service  = google_cloud_run_service.sdz_api[0].name
  location = google_cloud_run_service.sdz_api[0].location
  role     = "roles/run.invoker"
  member   = "allUsers"
}

resource "google_firebase_web_app" "sdz_web_app" {
  provider     = google-beta
  project      = var.sdz_project_id
  display_name = var.sdz_web_app_display_name

  depends_on = [google_firebase_project.sdz_firebase]
}

resource "google_firebase_android_app" "sdz_android_app" {
  provider     = google-beta
  project      = var.sdz_project_id
  display_name = var.sdz_android_app_display_name
  package_name = var.sdz_android_package_name

  count      = var.sdz_android_package_name == "" ? 0 : 1
  depends_on = [google_firebase_project.sdz_firebase]
}

resource "google_firebase_apple_app" "sdz_ios_app" {
  provider     = google-beta
  project      = var.sdz_project_id
  display_name = var.sdz_ios_app_display_name
  bundle_id    = var.sdz_ios_bundle_id

  count      = var.sdz_ios_bundle_id == "" ? 0 : 1
  depends_on = [google_firebase_project.sdz_firebase]
}
