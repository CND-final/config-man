import { Module } from '@nestjs/common';
import { ProjectsModule } from '../projects/projects.module';
import { TemplatesModule } from '../templates/templates.module';
import { ValidationController } from './validation.controller';
import { ValidationService } from './validation.service';

@Module({
  imports: [ProjectsModule, TemplatesModule],
  controllers: [ValidationController],
  providers: [ValidationService],
  exports: [ValidationService]
})
export class ValidationModule {}
