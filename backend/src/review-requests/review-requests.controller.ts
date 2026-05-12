import {
  Body,
  Controller,
  Get,
  Headers,
  Param,
  Post,
  Put,
  Query
} from '@nestjs/common';
import { AuthService } from '../auth/auth.service';
import { CreateReviewRequestDto } from './dto/create-review-request.dto';
import { ReviewDecisionDto } from './dto/review-decision.dto';
import { ReviewRequestsService } from './review-requests.service';

@Controller()
export class ReviewRequestsController {
  constructor(
    private readonly authService: AuthService,
    private readonly reviewRequestsService: ReviewRequestsService
  ) {}

  @Get('review-requests')
  listAll(@Headers('authorization') authorization?: string) {
    this.authService.requireUser(authorization);
    return this.reviewRequestsService.listAll();
  }

  @Get('projects/:projectId/review-requests')
  listForProject(
    @Param('projectId') projectId: string,
    @Query('env') environment?: string,
    @Query('key') configKey?: string,
    @Query('status') status?: string,
    @Headers('authorization') authorization?: string
  ) {
    this.authService.requireUser(authorization);
    return this.reviewRequestsService.listForProject(projectId, {
      environment,
      configKey,
      status
    });
  }

  @Post('review-requests')
  create(
    @Body() dto: CreateReviewRequestDto,
    @Headers('authorization') authorization?: string
  ) {
    const user = this.authService.requireUser(authorization);
    return this.reviewRequestsService.create(user, dto);
  }

  @Put('review-requests/:requestId/approve')
  approve(
    @Param('requestId') requestId: string,
    @Body() dto: ReviewDecisionDto,
    @Headers('authorization') authorization?: string
  ) {
    const user = this.authService.requireUser(authorization);
    return this.reviewRequestsService.approve(user, requestId, dto.comment);
  }

  @Put('review-requests/:requestId/reject')
  reject(
    @Param('requestId') requestId: string,
    @Body() dto: ReviewDecisionDto,
    @Headers('authorization') authorization?: string
  ) {
    const user = this.authService.requireUser(authorization);
    return this.reviewRequestsService.reject(user, requestId, dto.comment);
  }
}
