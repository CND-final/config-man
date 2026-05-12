import { Injectable, NotFoundException } from '@nestjs/common';
import { ChangeRequestStatus, Prisma } from '@prisma/client';
import { AuthService } from '../auth/auth.service';
import { DemoUser } from '../auth/auth.types';
import { PrismaService } from '../prisma/prisma.service';
import { ProjectsService } from '../projects/projects.service';
import { CreateReviewRequestDto } from './dto/create-review-request.dto';

interface ReviewRequestFilters {
  environment?: string;
  configKey?: string;
  status?: string;
}

@Injectable()
export class ReviewRequestsService {
  constructor(
    private readonly prisma: PrismaService,
    private readonly authService: AuthService,
    private readonly projectsService: ProjectsService
  ) {}

  async listAll() {
    const requests = await this.prisma.changeRequest.findMany({
      orderBy: { createdAt: 'desc' },
      include: { project: true }
    });

    return requests.map((request) => this.serialize(request));
  }

  async listForProject(projectId: string, filters: ReviewRequestFilters) {
    await this.projectsService.ensureProjectExists(projectId);

    const where: Prisma.ChangeRequestWhereInput = {
      projectId,
      ...(filters.environment ? { environment: filters.environment } : {}),
      ...(filters.configKey ? { configKey: filters.configKey } : {}),
      ...(this.isStatus(filters.status)
        ? { status: filters.status as ChangeRequestStatus }
        : {})
    };

    const requests = await this.prisma.changeRequest.findMany({
      where,
      orderBy: { createdAt: 'desc' },
      include: { project: true }
    });

    return requests.map((request) => this.serialize(request));
  }

  async create(user: DemoUser, dto: CreateReviewRequestDto) {
    this.authService.requireReviewCreation(user);
    await this.projectsService.ensureEnvironmentExists(
      dto.projectId,
      dto.environment
    );

    const request = await this.prisma.$transaction(async (tx) => {
      const created = await tx.changeRequest.create({
        data: {
          projectId: dto.projectId,
          environment: dto.environment,
          configKey: dto.configKey,
          requester: user.name,
          reason: dto.reason
        },
        include: { project: true }
      });

      await tx.auditLog.create({
        data: {
          actor: user.name,
          action: 'review_request.create',
          resourceType: 'change_request',
          resourceId: created.id,
          projectId: dto.projectId,
          metadata: {
            environment: dto.environment,
            configKey: dto.configKey,
            reason: dto.reason
          }
        }
      });

      return created;
    });

    return this.serialize(request);
  }

  async approve(user: DemoUser, requestId: string, comment?: string) {
    return this.setStatus(user, requestId, 'approved', comment);
  }

  async reject(user: DemoUser, requestId: string, comment?: string) {
    return this.setStatus(user, requestId, 'rejected', comment);
  }

  private async setStatus(
    user: DemoUser,
    requestId: string,
    status: ChangeRequestStatus,
    comment?: string
  ) {
    this.authService.requireReviewPermission(user);

    const existing = await this.prisma.changeRequest.findUnique({
      where: { id: requestId }
    });

    if (!existing) {
      throw new NotFoundException(`Review request "${requestId}" not found`);
    }

    const request = await this.prisma.$transaction(async (tx) => {
      const updated = await tx.changeRequest.update({
        where: { id: requestId },
        data: {
          status,
          reviewer: user.name,
          comment
        },
        include: { project: true }
      });

      await tx.auditLog.create({
        data: {
          actor: user.name,
          action: `review_request.${status}`,
          resourceType: 'change_request',
          resourceId: requestId,
          projectId: updated.projectId,
          metadata: { comment }
        }
      });

      return updated;
    });

    return this.serialize(request);
  }

  private serialize(
    request: Prisma.ChangeRequestGetPayload<{ include: { project: true } }>
  ) {
    return {
      id: request.id,
      projectId: request.projectId,
      projectName: request.project.name,
      environment: request.environment,
      configKey: request.configKey,
      requester: request.requester,
      reviewer: request.reviewer,
      status: request.status,
      reason: request.reason,
      comment: request.comment,
      createdAt: request.createdAt,
      updatedAt: request.updatedAt
    };
  }

  private isStatus(status?: string) {
    return ['pending', 'approved', 'rejected'].includes(status ?? '');
  }
}
