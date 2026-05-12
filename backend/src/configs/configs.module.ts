import { Module } from '@nestjs/common';
import { AuthModule } from '../auth/auth.module';
import { ProjectsModule } from '../projects/projects.module';
import { ConfigsController } from './configs.controller';
import { ConfigsService } from './configs.service';

@Module({
  imports: [AuthModule, ProjectsModule],
  controllers: [ConfigsController],
  providers: [ConfigsService],
  exports: [ConfigsService]
})
export class ConfigsModule {}
