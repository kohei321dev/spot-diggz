# 5府県スケートパーク調査スナップショット

- 基準日: 2026-07-19
- 対象: 大阪府、兵庫県、和歌山県、奈良県、徳島県
- 公開カタログ: `data/facilities.json`
- 掲載保留台帳: `data/facility-candidates.json`

## 結論

自治体、指定管理者、施設運営者の公式情報を再監査し、31施設をカタログへ公開した。このうち大阪府は24施設で、原池公園、泉南りんくう公園、羽曳野、熊取、長居、二色の浜などを含む。日付別の一般利用予定が確認できない施設は、カタログには掲載するが `generalUseStatus=schedule_check_required` として推薦から除外する。別に13施設の存在を確認したが、通常利用の予約、営業時間、設備、ルール等の不足が残るため保留している。

この調査は確認できた公式Web情報のスナップショットであり、5府県内の全施設の完全網羅を保証しない。SNSだけで営業を告知する施設、路上スポット、開設計画、閉鎖を確認できない古い一覧だけに載る施設は公開対象外とした。

## 公開施設

| ID | 府県 | 施設 | 代表一次情報 | 採用根拠の要約 |
| --- | --- | --- | --- | --- |
| OSK-F001 | 大阪府 | SPOTAKA Skateboard Park（スポパー） | [施設運営者](https://spotaka-skateboardpark.com/) | 営業時間、料金、登録、年齢ルール、初心者向け設備、座標を確認 |
| OSK-F002 | 大阪府 | スポーツパークまつばら | [指定管理者](https://shisetsu.mizuno.jp/skatepark/sportspark-matsubara) | 公式情報間の終了時刻差を安全側の22:00で採用し、年末年始休場を構造化 |
| OSK-F003 | 大阪府 | おくさま印スケボーパーク | [指定管理者](https://shisetsu.mizuno.jp/m-7288/guide) | 屋内外料金、登録、年齢・防具ルール、アクセスを確認 |
| OSK-F004 | 大阪府 | 箕面スケートボードパーク | [指定管理者](https://shisetsu.mizuno.jp/skatepark/minoh) | 第4木曜休場、料金、登録、初心者エリア、レンタル、座標を確認 |
| HYG-F001 | 兵庫県 | 三木スケートボードパーク | [三木市](https://www.city.miki.lg.jp/site/skate-park/index.html) | 日没終了、木曜休場、料金、登録、防具・年齢ルール、初心者教室を確認 |
| HYG-F002 | 兵庫県 | でんすぽ スケートパーク | [指定管理者](https://shisetsu.mizuno.jp/skatepark/denspo) | 第4火曜休場、料金、受付、経験者向け設備、座標を確認 |
| WAK-F001 | 和歌山県 | GOBOスケートパーク | [御坊市](https://www.city.gobo.lg.jp/sosiki/kyoikuiin/kyoiku/tanto/sports_recreation/8679.html) | 営業時間、無料・年度登録、照明、初心者教室、年齢・防具ルールを確認 |
| WAK-F002 | 和歌山県 | 和歌山市つつじが丘総合公園 スケートボード場 | [和歌山市](https://www.city.wakayama.wakayama.jp/shisetsu/kouen_sp_shisetsu/1006086/1060589.html) | 季節営業時間、年度料金・登録、年齢、初心者設備・レンタルを確認 |
| WAK-F003 | 和歌山県 | 大宮緑地総合運動公園 スケートボード場 | [岩出市](https://www.city.iwade.lg.jp/soshiki/20/2493.html) | 営業時間、無料・事前申請、防具を確認。公式情報にないbank/railは登録しない |
| NAR-F001 | 奈良県 | ロートスケートボードパーク奈良 | [奈良市](https://www.city.nara.lg.jp/soshiki/22/182822.html) / [内閣府の2026年祝日](https://www8.cao.go.jp/chosei/shukujitsu/gaiyou.html) | 水曜・祝日翌日・年末年始休場を確認。海の日翌日の7月21日と、山の日翌日かつ水曜の8月12日を構造化 |
| TKS-F001 | 徳島県 | おきのすインドアパーク スケートボードパーク | [施設運営者](https://oipark.jp/indoor/skateboard-park/) | 屋上の屋外施設、天候不良時の中断・中止、第4水曜・年末年始休館、区分料金、誓約書、初心者用設備を確認 |

### 大阪府再監査で追加した公共施設

| ID | 施設 | 一次情報 | 一般利用の扱い |
| --- | --- | --- | --- |
| OSK-F005 | 服部緑地スケートボードパーク | [服部緑地](https://hattori-ryokuchi.com/%E3%82%B9%E3%82%B1%E3%83%BC%E3%83%88%E3%83%9C%E3%83%BC%E3%83%89%E3%83%91%E3%83%BC%E3%82%AF/) | 通常利用 |
| OSK-F006 | 深北緑地 波の広場 | [指定管理者](https://shisetsu.mizuno.jp/skatepark/fukakita-ryokuchi) | 通常利用 |
| OSK-F007 | 久宝寺緑地 スケートボードエリア | [久宝寺緑地](https://kyuhoji-ryokuchi.jp/guide.html) | 公園時間を安全側に表示 |
| OSK-F008 | 大泉緑地 スケート広場 | [大泉緑地](https://oizumi.osaka-park.or.jp/facility/581/) | 日付別確認が必要 |
| OSK-F009 | りんくう公園 RinX-SQUARE | [りんくう公園](https://rinku.osaka-park.or.jp/20872) | 日付別確認が必要 |
| OSK-F010 | 二色の浜公園 スケートパーク | [二色の浜公園](https://nishikinohama-park.com/facility/sports/) | 通常利用 |
| OSK-F011 | タイガーラックスケートボードパーク長居 | [長居公園](https://nagaipark.com/guide/skateboard/) | 通常利用 |
| OSK-F012 | 原池公園スケートボードパーク | [指定管理者](https://shisetsu.mizuno.jp/skatepark/baraike) | 通常利用。定休日・年齢ルールを構造化 |
| OSK-F013 | まなび中央公園 スケートボード広場 | [施設運営者](https://kishiwada-hotparks.jp/manabichuopark/exercise/) | 夜間禁止・安全側表示 |
| OSK-F014 | 今池公園 スケートボード広場 | [岸和田市](https://www.city.kishiwada.osaka.jp/soshiki/132/kouen-ichiran.html) | 日付別確認が必要 |
| OSK-F015 | 永楽ゆめの森公園スケートパーク | [施設運営者](https://kumatori-eirakupark.net/yugu/) | 季節時間を反映 |
| OSK-F016 | 泉南りんくう公園 スケートパーク | [泉南りんくう公園](https://sennanlongpark.com/?page_id=357) | 日の出・日没の確認が必要 |
| OSK-F017 | 総建スケボーパークはびきの | [羽曳野市](https://www.city.habikino.lg.jp/soshiki/doboku/dourokouen/kouennkankei/tokusyokunoarukouen/15143.html) | 季節時間を反映 |
| OSK-F018 | 泉大津円形スケートパーク | [泉大津市](https://www.city.izumiotsu.lg.jp/kosodate/odekake/park/11132.html) | 日の出・日没の確認が必要 |
| OSK-F019 | マンデー末広公園山側スケートボードパーク | [泉佐野市](https://www.city.izumisano.lg.jp/kakuka/toshi/doro/menu/kouen/10036.html) | 通常利用 |
| OSK-F020 | 王仁公園スケートボード広場 | [枚方市](https://www.city.hirakata.osaka.jp/0000053913.html) | 2026年7月の季節時間を反映 |
| OSK-F021 | 福万寺町市民運動広場南面 スケートボード場 | [八尾市](https://www.city.yao.osaka.jp/bunka_sports_event/sports/1011000/1011006/1011013.html) | 季節時間・治水緑地の制約を反映 |
| OSK-F022 | 高師浜総合運動施設 スケートボード場 | [高石市](https://www.city.takaishi.lg.jp/kakuka/kyouiku/syakaikyouiku__ka/lifelongstudy/undoushisetsu/hokashisetsu.html) | 申請・有料利用 |
| OSK-F023 | 倉治1丁目高架下スケボー広場 | [交野市](https://www.city.katano.osaka.jp/docs/2025031800021/) | 日付別確認が必要 |
| OSK-F024 | せんなん里海公園 PLAYGROUND AREA | [大阪府営公園](https://sennan.osaka-park.or.jp/2024/10/06/playground-area%EF%BC%863on3%E3%83%90%E3%82%B9%E3%82%B1%E3%83%83%E3%83%88%E3%82%B3%E3%83%BC%E3%83%88/) | 日付別確認が必要 |

## 保留施設

保留13件の機械可読な一覧、再確認URL、掲載を止めた具体的な不足項目は `data/facility-candidates.json` に保存した。原池、深北緑地、長居、服部緑地、永楽、泉南りんくう公園は今回の再監査で保留から公開へ移した。主な保留理由は次のとおり。

- 施設単体の座標を公式地図から確認できない
- 定休日、年齢、料金が同じ運営主体のページ間で一致しない
- 通常利用の登録・予約要否、営業時間、防具ルールが明記されていない
- 初心者向けまたは経験者向けである根拠がない
- 2026年の改修後情報や臨時営業情報がSNSにしかない
- 複数エリアごとに営業時間が異なり、現在の施設単位schemaへ安全に集約できない

## データ化方針

- `verifiedAt`、`dynamicVerifiedAt`、`stableVerifiedAt` は2026-07-19の調査スナップショット時刻を記録する。
- 営業時間、料金、休場、予約手続は30日、住所、座標、設備、主要ルールは180日を鮮度目標とする。
- 年末年始は `annual`、基準日から30日以内に発生する第4曜日休場は `one_time` として推薦除外に使う。
- 祝日翌日休場は施設の公式休場規則と内閣府の祝日一覧を突き合わせ、対象日を `one_time` として登録する。同じ日が通常休場日でも、根拠と再確認対象を明確にするため保持する。
- 季節時間や日没終了は、推薦判定では公式情報を超えない安全側の時刻を採用し、元の条件を `scheduleNotes` に残す。
- 施設固有の営業時間または日付別の一般利用予定が確認できない場合は `generalUseStatus=schedule_check_required` とし、施設一覧の参照対象にはするが推薦対象から除外する。スクール・貸切で一般滑走枠が変動するSPOTAKAも同じ扱いとする。
- 日本語の事実情報を短く言い換え、英訳を併記する。公式ページの写真、ロゴ、長い説明文は複製しない。
- 出発地点の正確な座標は施設データに含めず、リクエスト処理後に保存しない。

## 再確認

動的情報は2026-08-18まで、安定情報は2027-01-15までを目安に再確認する。期限を超えた施設は推薦エンジンが除外する。第4曜日の定例休場は現schemaで無期限に表現できないため、動的情報の再確認時に次の休場日を更新する。

`make verify-catalog` はproduction catalogが実行時点から168時間後も鮮度内であることを検査する。GitHub Actionsでも毎週実行し、再確認期限の7日前までに失敗させる。このcheckは公式情報の再調査そのものではないため、失敗時は各 `sourceUrl` を確認し、確認できた属性だけを更新する。`testdata/facilities.dev.json` などの固定fixtureはこの運用判定に使用しない。
