import { Module } from '@nestjs/common';
import { ProjectsModule } from '../projects/projects.module';
import { ConfigsController } from './configs.controller';
import { ConfigsService } from './configs.service';

@Module({
  imports: [ProjectsModule],
  controllers: [ConfigsController],
  providers: [ConfigsService],
  exports: [ConfigsService]
})
export class ConfigsModule {}
