import { Module } from '@nestjs/common';
import { AdminExtrasController } from './admin-extras.controller';
import { AdminExtrasService } from './admin-extras.service';
import { DatabaseModule } from '../../common/database.module';

@Module({
  imports: [DatabaseModule],
  controllers: [AdminExtrasController],
  providers: [AdminExtrasService],
})
export class AdminExtrasModule {}
