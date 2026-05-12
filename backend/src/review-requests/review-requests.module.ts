import { Module } from '@nestjs/common';
import { AuthModule } from '../auth/auth.module';
import { ProjectsModule } from '../projects/projects.module';
import { ReviewRequestsController } from './review-requests.controller';
import { ReviewRequestsService } from './review-requests.service';

@Module({
  imports: [AuthModule, ProjectsModule],
  controllers: [ReviewRequestsController],
  providers: [ReviewRequestsService],
  exports: [ReviewRequestsService]
})
export class ReviewRequestsModule {}
