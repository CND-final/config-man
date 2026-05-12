import { Module } from '@nestjs/common';
import { AuditModule } from './audit/audit.module';
import { AuthModule } from './auth/auth.module';
import { ConfigsModule } from './configs/configs.module';
import { HealthController } from './health.controller';
import { PrismaModule } from './prisma/prisma.module';
import { ProjectsModule } from './projects/projects.module';
import { ReviewRequestsModule } from './review-requests/review-requests.module';
import { TemplatesModule } from './templates/templates.module';
import { ValidationModule } from './validation/validation.module';

@Module({
  imports: [
    PrismaModule,
    AuthModule,
    AuditModule,
    TemplatesModule,
    ProjectsModule,
    ConfigsModule,
    ReviewRequestsModule,
    ValidationModule
  ],
  controllers: [HealthController]
})
export class AppModule {}
