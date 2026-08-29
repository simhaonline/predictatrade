import { Body, Controller, Get, Post, Put, Param, UseGuards } from '@nestjs/common';
import { LicensingService } from './licensing.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { RolesGuard, Roles, Role, PermissionGuard, Permission, RequirePermissions } from '../../common/guards/roles.guard';
import { CurrentUser } from '../../common/decorators/current-user.decorator';

@Controller('licensing')
export class LicensingController {
  constructor(private licensingService: LicensingService) {}

  // Public endpoint — no JWT required (used by Windows Agent with license key only)
  @Post('validate')
  async validateLicense(@Body() body: { license_key?: string }) {
    if (!body.license_key) {
      return { valid: false, status: 'NO_KEY', error: 'License key is required' };
    }
    return this.licensingService.validateLicenseKey(body.license_key);
  }

  // All endpoints below require JWT authentication
  @UseGuards(JwtAuthGuard)
  @Get('licenses')
  async listLicenses(@CurrentUser('sub') userId: string) {
    return this.licensingService.listLicenses(userId);
  }

  @UseGuards(JwtAuthGuard)
  @Get('devices')
  async listDevices(@CurrentUser('sub') userId: string) {
    return this.licensingService.listDevices(userId);
  }

  @UseGuards(JwtAuthGuard)
  @Post('devices')
  async registerDevice(@CurrentUser('sub') userId: string, @Body() body: any) {
    return this.licensingService.registerDevice(userId, body);
  }

  @UseGuards(JwtAuthGuard)
  @Get('mt-accounts')
  async listMtAccounts(@CurrentUser('sub') userId: string) {
    return this.licensingService.listMtAccounts(userId);
  }

  // Admin-wide listing (check.md #4): the user-scoped endpoint returns empty
  // for admin sessions; admins need the fleet view (all linked MT accounts).
  @UseGuards(JwtAuthGuard, RolesGuard, PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @Get('admin-mt-accounts')
  async listAllMtAccounts() {
    return this.licensingService.listAllMtAccounts();
  }

  @UseGuards(JwtAuthGuard)
  @Post('mt-accounts')
  async registerTerminal(@CurrentUser('sub') userId: string, @Body() body: any) {
    return this.licensingService.registerTerminal(userId, body);
  }

  @UseGuards(JwtAuthGuard)
  @Put('devices/:id/heartbeat')
  async heartbeat(@CurrentUser('sub') userId: string, @Param('id') deviceId: string, @Body() body: any) {
    // P0-CP4 fix: ownership scoping — users may only heartbeat their own devices
    return this.licensingService.heartbeat(deviceId, body, userId);
  }

  @UseGuards(JwtAuthGuard)
  @Post('devices/:id/revoke')
  async revokeDevice(@CurrentUser('sub') userId: string, @Param('id') deviceId: string, @Body() body: any) {
    // P0-CP4 fix: ownership scoping (admins use the admin endpoint)
    return this.licensingService.revokeDevice(deviceId, body.reason || 'User revoked', userId);
  }

  @UseGuards(JwtAuthGuard)
  @Post('terminals/sync')
  async syncTerminalAccount(@CurrentUser('sub') userId: string, @Body() body: any) {
    return this.licensingService.syncTerminalAccount(userId, body);
  }

  // ============================================================
  // Admin-only license lifecycle endpoints
  // ============================================================
  @UseGuards(JwtAuthGuard, RolesGuard, PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @RequirePermissions(Permission.LICENSE_MANAGE)
  @Post('licenses')
  async createLicense(@Body() body: {
    user_id: string; plan_id: string; max_devices?: number; max_mt_accounts?: number;
    allowed_strategies?: string[]; allowed_execution_modes?: string[]; valid_days?: number;
  }) {
    return this.licensingService.createLicense(body);
  }

  @UseGuards(JwtAuthGuard, RolesGuard, PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @RequirePermissions(Permission.LICENSE_MANAGE)
  @Post('licenses/:id/suspend')
  async suspendLicense(@Param('id') id: string, @Body() body: { reason?: string }) {
    return this.licensingService.suspendLicense(id, body?.reason || 'Admin suspended');
  }

  @UseGuards(JwtAuthGuard, RolesGuard, PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @RequirePermissions(Permission.LICENSE_MANAGE)
  @Post('licenses/:id/revoke')
  async revokeLicense(@Param('id') id: string, @Body() body: { reason?: string }) {
    return this.licensingService.revokeLicense(id, body?.reason || 'Admin revoked');
  }

  @UseGuards(JwtAuthGuard, RolesGuard, PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @RequirePermissions(Permission.LICENSE_MANAGE)
  @Post('licenses/:id/renew')
  async renewLicense(@Param('id') id: string, @Body() body: { valid_days?: number }) {
    return this.licensingService.renewLicense(id, body?.valid_days);
  }

  @UseGuards(JwtAuthGuard, RolesGuard, PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @RequirePermissions(Permission.LICENSE_MANAGE)
  @Post('licenses/:id/reset')
  async resetLicense(@Param('id') id: string) {
    return this.licensingService.resetLicense(id);
  }

  @UseGuards(JwtAuthGuard, RolesGuard, PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @RequirePermissions(Permission.LICENSE_MANAGE)
  @Post('licenses/:id/force-logout')
  async forceLogoutLicense(@Param('id') id: string) {
    return this.licensingService.forceLogoutLicense(id);
  }

  @UseGuards(JwtAuthGuard, RolesGuard, PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @RequirePermissions(Permission.LICENSE_MANAGE)
  @Get('licenses/:id/activations')
  async fetchLicenseActivations(@Param('id') id: string) {
    return this.licensingService.fetchLicenseActivations(id);
  }

  // ============================================================
  // Admin-only device security-action endpoints
  // ============================================================
  @UseGuards(JwtAuthGuard, RolesGuard, PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @RequirePermissions(Permission.LICENSE_MANAGE)
  @Post('devices/:id/reset')
  async resetDevice(@Param('id') id: string) {
    return this.licensingService.resetDevice(id);
  }

  @UseGuards(JwtAuthGuard, RolesGuard, PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @RequirePermissions(Permission.LICENSE_MANAGE)
  @Post('devices/:id/force-upgrade')
  async forceUpgradeDevice(@Param('id') id: string) {
    return this.licensingService.forceUpgradeDevice(id);
  }

  @UseGuards(JwtAuthGuard, RolesGuard, PermissionGuard)
  @Roles(Role.ADMIN, Role.SUPER_ADMIN)
  @RequirePermissions(Permission.LICENSE_MANAGE)
  @Post('devices/:id/disable-signal')
  async disableDeviceSignal(@Param('id') id: string) {
    return this.licensingService.disableDeviceSignal(id);
  }
}
