# Admin Backend Gap Report

## Gaps Identified

1. **Service Start/Stop/Restart**: The backend does not expose systemd service control endpoints. The operations controller handles trading halt/resume and signal pause/resume, but does NOT start/stop/restart system services (NestJS, Go engine, PostgreSQL, Valkey). This is by design — service management should be done via SSH/systemd, not through the browser.

2. **User Approval Workflow**: The backend supports status changes (ACTIVE, SUSPENDED, LOCKED, DELETED) but does not have a formal "pending approval" registration flow. New users are created as ACTIVE by default. The admin can suspend users after registration.

3. **Plan CRUD**: The backend exposes GET /plans and GET /plans/:id but does not expose POST/PUT/DELETE for plan management. Plans are managed via database seeds.

4. **License CRUD**: The backend exposes GET /admin/licenses (admin list) but does not expose create/revoke/extend endpoints. License management is handled via the device activation flow.

5. **Invoice Generation**: The backend exposes GET /billing/invoices (user's own invoices) but does not have an admin endpoint for invoice management or generation. Invoices are created by the billing webhook.

6. **Avatar Upload**: The backend does not expose avatar upload/storage. Profile photos are not currently supported.

7. **Admin Invoice List**: There is no admin-specific invoice endpoint. Admin billing uses subscription/commission/payout data instead.

## Conclusion

All gaps are documented backend limitations, not frontend omissions. The frontend correctly integrates every endpoint the backend exposes. No mock data is used to fill gaps.
