# Device Account Type Design

## Goal

Add an administrator-managed device capability that classifies QQ cache accounts by account type. Administrators can filter and export QQ cache records by type. Sales extraction is limited by a global administrator setting of allowed account types.

The change must be non-destructive:

- Existing online QQ cache data is treated as the default account type.
- Unconfigured devices continue to produce default account type records.
- Device configuration changes only affect future uploads and imports.
- Administrators remain unrestricted when exporting QQ cache data.

## Account Types

The first version supports two account types:

- `default`: existing/default account type.
- `pc`: PC account type.

The backend owns the enum and exposes it to the frontend so labels stay consistent across Device Management, QQ Cache Management, and Sales Extract.

## Device Management

Add a new independent administrator-only menu named `设备管理`. It is separate from the existing Redis-based device heartbeat/busy-state service. This menu is a persistent configuration surface and does not show or mutate online device state.

The first version manages:

- device ID.
- account type.
- optional remark.

Suggested table:

```text
sys_devices
id
device_id       unique, required
account_type    default / pc, required
remark
created_at
updated_at
deleted_at
```

Backend behavior:

- Only administrator roles can access device CRUD APIs: role `100` and role `888`.
- Sales and upload clients cannot view or modify device configuration.
- `ResolveAccountType(deviceID)` returns:
  - configured account type if the device exists and the value is valid.
  - `default` when the device ID is empty, missing, deleted, or invalid.
- QQ cache services only depend on `ResolveAccountType`; they do not own device CRUD.

Frontend behavior:

- New `设备管理` menu is assigned only to administrator roles.
- List supports searching by device ID and account type.
- Add/edit uses an account type dropdown.
- Delete is a soft delete.

## QQ Cache Record Changes

Add `account_type` to QQ cache records:

```text
qq_cache_records.account_type  default / pc, indexed
```

Write behavior:

- App upload, phone-register upload, internal tool import, and admin import resolve account type from `deviceId` and write it into the QQ cache record.
- Upsert/update of an existing QQ number updates `account_type` using the current upload/import `deviceId`, matching the existing behavior that replaces phone, password, INI, client version, and device ID.
- Device configuration changes do not backfill existing QQ cache records.

Read compatibility:

- `NULL` or empty `account_type` is treated as `default`.
- Admin filters for `default` include `account_type IS NULL`, `account_type = ''`, and `account_type = 'default'`.

## QQ Cache Management

Admin QQ Cache Management adds an account type filter and table column.

Affected admin operations:

- List supports `accountType`.
- Account list export supports `accountType` when exporting by current filters.
- Pending INI extraction by count supports `accountType` as an optional filter.
- Selected-row export continues to use selected IDs and remains unrestricted.
- TXT-based export by QQ list remains unrestricted because it is explicitly account-driven; it can export matching accounts regardless of type.

Administrators and super administrators are not limited by sales allowed-type configuration.

## Sales Export Configuration

Add a global QQ cache setting for sales allowed export account types:

```json
{
  "salesAllowedAccountTypes": ["default", "pc"]
}
```

This setting is managed in admin QQ Cache Management because it controls QQ cache sales extraction.

Rules:

- Empty or missing configuration means sales cannot export any account type.
- The setting is global for all sales users.
- Invalid account type values are ignored.
- The admin UI should make an empty setting explicit.

Store this setting in the existing `sys_params` table to avoid adding another configuration table. Use a dedicated key such as `qq_cache_sales_allowed_account_types`, with the value stored as a JSON string array. Missing row, empty value, invalid JSON, or an empty parsed array all mean sales cannot export any account type.

## Sales Extraction

Sales-facing endpoints must apply the global allowed-type filter:

- `GET /qqCache/sales/summary`
- `POST /qqCache/sales/extract`

Behavior:

- If no allowed account type is configured:
  - summary returns zero available count.
  - extract fails with a clear message such as `未配置可导出账号类型`.
- If allowed types are configured:
  - summary counts only pending records whose normalized account type is allowed.
  - extract locks and exports only pending records whose normalized account type is allowed.
- Existing recent-minute filters still apply after the allowed-type constraint.

Sales history and settlement remain based on actual extracted batches and records. They do not need to be hidden by current allowed-type changes, because changing allowed types later must not erase past extraction history.

## API Shape

Device module:

```text
POST /device/list
POST /device/save
POST /device/delete
GET  /device/accountTypes
```

QQ cache additions:

```text
GET  /qqCache/accountTypes
GET  /qqCache/salesAllowedAccountTypes
POST /qqCache/salesAllowedAccountTypes
```

Existing QQ cache requests add optional `accountType` where filter-driven exports or lists are used:

```text
QQCacheList.accountType
QQCacheExportPendingIniZip.accountType
QQCacheExportAccountList.accountType
```

Sales extract does not accept arbitrary requested account types in the first version. It uses the global allowed set only.

## Permissions

- Device Management APIs: administrator and super administrator only.
- Device Management menu: administrator and super administrator only.
- QQ cache sales allowed-type configuration: administrator and super administrator only.
- Admin QQ cache list/export: administrator and super administrator only, unrestricted by sales config.
- Sales extract: sales role only, restricted by global allowed types.

## Migration

Use additive migrations only:

- Add `account_type` to QQ cache records with a default of `default` where supported.
- Add an index on `account_type`.
- Add `sys_devices`.

No data rewrite is required for existing QQ cache rows. Runtime query compatibility treats blank values as `default`.

## Testing

Backend tests should cover:

- `ResolveAccountType` returns configured type and falls back to `default`.
- QQ cache upload/import writes `default` for unconfigured devices.
- QQ cache upload/import writes `pc` for configured PC devices.
- Existing blank account type matches admin `default` filters.
- Admin list/export filters by account type.
- Sales summary returns zero when no allowed types are configured.
- Sales extract fails when no allowed types are configured.
- Sales summary/extract only include globally allowed account types.
- Admin exports remain unaffected by sales allowed-type configuration.

Frontend verification should cover:

- Device Management menu visibility for admin roles only.
- Device create/edit/delete flow.
- QQ cache account type filter and column.
- Sales allowed-type configuration save/load.
- Sales extract page shows zero available when config is empty.

## Out of Scope

- Per-sales-user account type permissions.
- Dynamic reclassification of historical QQ cache records when a device config changes.
- Merging persistent Device Management with Redis online/busy device state.
- Backfilling existing QQ cache records from historical device IDs.
