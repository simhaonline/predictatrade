import { Module } from '@nestjs/common';
import { OperationsController } from './operations.controller';
import { AiProvidersController, AiProvidersService } from './ai-providers.service';
import { OperationsService } from './operations.service';
import { DatabaseModule } from '../../common/database.module';

@Module({
  imports: [DatabaseModule],
  controllers: [OperationsController, AiProvidersController],
  providers: [OperationsService, AiProvidersService],
})
export class OperationsModule {}
