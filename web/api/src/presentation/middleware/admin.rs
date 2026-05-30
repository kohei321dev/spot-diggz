use axum::extract::FromRequestParts;
use axum::http::request::Parts;

use crate::presentation::error::SdzApiError;

use super::auth::SdzAuthUser;

#[derive(Debug, Clone)]
pub struct SdzAdminUser {
    pub sdz_user_id: String,
}

impl<S> FromRequestParts<S> for SdzAdminUser
where
    S: Send + Sync,
{
    type Rejection = SdzApiError;

    #[allow(clippy::manual_async_fn)]
    fn from_request_parts(
        parts: &mut Parts,
        state: &S,
    ) -> impl std::future::Future<Output = Result<Self, Self::Rejection>> + Send {
        async move {
            let auth_user = SdzAuthUser::from_request_parts(parts, state).await?;

            let admin_uids = std::env::var("SDZ_ADMIN_UIDS").unwrap_or_default();
            let is_admin = admin_uids
                .split(',')
                .any(|uid| uid.trim() == auth_user.sdz_user_id);

            if !is_admin {
                tracing::warn!(
                    event_code = "SDZ-API-1010",
                    component = "middleware",
                    user_id = %auth_user.sdz_user_id,
                    "admin access denied"
                );
                return Err(SdzApiError::Forbidden("admin access required".to_string()));
            }

            tracing::info!(
                event_code = "SDZ-API-1011",
                component = "middleware",
                user_id = %auth_user.sdz_user_id,
                "admin access granted"
            );

            Ok(SdzAdminUser {
                sdz_user_id: auth_user.sdz_user_id,
            })
        }
    }
}
