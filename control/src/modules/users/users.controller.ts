import { Body, Controller, Delete, Get, Param, Patch, Query, UseGuards, Post } from '@nestjs/common';
import { UsersService } from './users.service';
import { JwtAuthGuard } from '../../common/guards/jwt-auth.guard';
import { AdminGuard } from '../../common/guards/admin.guard';
import { CurrentUser } from '../../common/decorators/current-user.decorator';
import { UpdateUserDto } from './dto/update-user.dto';

@Controller('users')
@UseGuards(JwtAuthGuard)
export class UsersController {
  constructor(private usersService: UsersService) {}

  @Get('me')
  async getMe(@CurrentUser('sub') userId: string) {
    return this.usersService.findById(userId);
  }

  @Patch('me')
  async updateMe(@CurrentUser('sub') userId: string, @Body() dto: UpdateUserDto) {
    return this.usersService.update(userId, dto);
  }

  @UseGuards(AdminGuard)
  @Get(':id')
  async findById(@Param('id') id: string) {
    return this.usersService.findById(id);
  }

  @UseGuards(AdminGuard)
  @Get()
  async list(@Query('page') page?: string, @Query('limit') limit?: string) {
    return this.usersService.list(
      page ? parseInt(page, 10) : 1,
      limit ? parseInt(limit, 10) : 20,
    );
  }

}
// ─── Admin user management (check.md 2026-08-30 #7: Edit/Delete/Update) ───
@Controller('users')
@UseGuards(JwtAuthGuard, AdminGuard)
export class AdminUsersController {
  constructor(private usersService: UsersService) {}

  @UseGuards(AdminGuard)
  @Patch(':id/status')
  async setStatus(
    @Param('id') id: string,
    @Body() body: { status: 'ACTIVE' | 'PENDING' | 'SUSPENDED' | 'DELETED'; reason?: string },
  ) {
    return this.usersService.setStatus(id, body.status, body.reason || 'admin_action');
  }

  @UseGuards(AdminGuard)
  @Patch(':id/role')
  async setRole(
    @Param('id') id: string,
    @Body() body: { role: 'ADMIN' | 'SUPER_ADMIN' | 'USER' | 'RISK_MANAGER' | 'SUPPORT' | 'ANALYST' | 'AUDITOR' | 'TRADING_OPERATOR' },
  ) {
    return this.usersService.setRole(id, body.role);
  }

  @UseGuards(AdminGuard)
  @Patch(':id')
  async editUser(
    @Param('id') id: string,
    @Body() body: { displayName?: string; email?: string },
  ) {
    return this.usersService.editUser(id, body.displayName, body.email);
  }

  @UseGuards(AdminGuard)
  @Delete(':id')
  async deleteUser(@Param('id') id: string) {
    return this.usersService.deleteUser(id);
  }
}
