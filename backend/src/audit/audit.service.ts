import { Injectable } from '@nestjs/common';
import { Prisma } from '@prisma/client';
import { PrismaService } from '../prisma/prisma.service';

export interface AuditRecord {
  actor: string;
  action: string;
  resourceType: string;
  resourceId?: string;
  projectId?: string;
  metadata?: Prisma.InputJsonValue;
}

@Injectable()
export class AuditService {
  constructor(private readonly prisma: PrismaService) {}

  async record(record: AuditRecord) {
    return this.prisma.auditLog.create({
      data: {
        actor: record.actor,
        action: record.action,
        resourceType: record.resourceType,
        resourceId: record.resourceId,
        projectId: record.projectId,
        metadata: record.metadata
      }
    });
  }
}
