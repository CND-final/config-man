import { Module } from '@nestjs/common';
import { AuditModule } from './audit/audit.module';
import { ConfigsModule } from './configs/configs.module';
import { HealthController } from './health.controller';
import { PrismaModule } from './prisma/prisma.module';
import { ProjectsModule } from './projects/projects.module';
import { TemplatesModule } from './templates/templates.module';
import { ValidationModule } from './validation/validation.module';

@Module({
  imports: [
    PrismaModule,
    AuditModule,
    TemplatesModule,
    ProjectsModule,
    ConfigsModule,
    ValidationModule
  ],
  controllers: [HealthController]
})
export class AppModule {}
