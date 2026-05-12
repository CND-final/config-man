import {
  BadRequestException,
  ConflictException,
  Injectable,
  NotFoundException
} from '@nestjs/common';
import { ConfigEntry, Prisma } from '@prisma/client';
import { PrismaService } from '../prisma/prisma.service';
import { ProjectsService } from '../projects/projects.service';
import { CreateConfigDto } from './dto/create-config.dto';
import { UpdateConfigDto } from './dto/update-config.dto';

const MASKED_VALUE = '******';

@Injectable()
export class ConfigsService {
  constructor(
    private readonly prisma: PrismaService,
    private readonly projectsService: ProjectsService
  ) {}

  async listConfigs(
    projectId: string,
    environment: string,
    revealSensitive = false
  ) {
    if (!environment) {
      throw new BadRequestException('Query parameter "env" is required');
    }

    await this.projectsService.ensureEnvironmentExists(projectId, environment);

    const entries = await this.prisma.configEntry.findMany({
      where: { projectId, environment },
      orderBy: { key: 'asc' }
    });

    return {
      projectId,
      environment,
      entries: entries.map((entry) =>
        this.serializeConfigEntry(entry, revealSensitive)
      )
    };
  }

  async createConfig(projectId: string, dto: CreateConfigDto, actor: string) {
    await this.projectsService.ensureEnvironmentExists(
      projectId,
      dto.environment
    );

    try {
      const entry = await this.prisma.$transaction(async (tx) => {
        const created = await tx.configEntry.create({
          data: {
            projectId,
            environment: dto.environment,
            key: dto.key,
            value: dto.value,
            valueType: dto.valueType ?? 'string',
            isSensitive: dto.isSensitive ?? false,
            updatedBy: actor
          }
        });

        await tx.configVersion.create({
          data: {
            configId: created.id,
            oldValue: null,
            newValue: created.value,
            changedBy: actor,
            changeReason: dto.changeReason ?? 'create config'
          }
        });

        await tx.auditLog.create({
          data: {
            actor,
            action: 'config.create',
            resourceType: 'config_entry',
            resourceId: created.id,
            projectId,
            metadata: {
              environment: created.environment,
              key: created.key,
              isSensitive: created.isSensitive
            }
          }
        });

        return created;
      });

      return this.serializeConfigEntry(entry);
    } catch (error) {
      if (this.isUniqueConstraintError(error)) {
        throw new ConflictException(
          `Config key "${dto.key}" already exists in "${dto.environment}"`
        );
      }
      throw error;
    }
  }

  async updateConfig(
    projectId: string,
    configId: string,
    dto: UpdateConfigDto,
    actor: string
  ) {
    if (
      dto.value === undefined &&
      dto.valueType === undefined &&
      dto.isSensitive === undefined
    ) {
      throw new BadRequestException('No config fields provided for update');
    }

    const updated = await this.prisma.$transaction(async (tx) => {
      const existing = await tx.configEntry.findFirst({
        where: { id: configId, projectId }
      });

      if (!existing) {
        throw new NotFoundException(
          `Config "${configId}" not found for project "${projectId}"`
        );
      }

      const entry = await tx.configEntry.update({
        where: { id: configId },
        data: {
          value: dto.value ?? existing.value,
          valueType: dto.valueType ?? existing.valueType,
          isSensitive: dto.isSensitive ?? existing.isSensitive,
          updatedBy: actor
        }
      });

      await tx.configVersion.create({
        data: {
          configId,
          oldValue: existing.value,
          newValue: entry.value,
          changedBy: actor,
          changeReason: dto.changeReason ?? 'update config'
        }
      });

      await tx.auditLog.create({
        data: {
          actor,
          action: 'config.update',
          resourceType: 'config_entry',
          resourceId: configId,
          projectId,
          metadata: {
            environment: entry.environment,
            key: entry.key,
            valueType: entry.valueType,
            isSensitive: entry.isSensitive
          }
        }
      });

      return entry;
    });

    return this.serializeConfigEntry(updated);
  }

  async deleteConfig(projectId: string, configId: string, actor: string) {
    const deleted = await this.prisma.$transaction(async (tx) => {
      const existing = await tx.configEntry.findFirst({
        where: { id: configId, projectId }
      });

      if (!existing) {
        throw new NotFoundException(
          `Config "${configId}" not found for project "${projectId}"`
        );
      }

      await tx.configEntry.delete({ where: { id: configId } });

      await tx.auditLog.create({
        data: {
          actor,
          action: 'config.delete',
          resourceType: 'config_entry',
          resourceId: configId,
          projectId,
          metadata: {
            environment: existing.environment,
            key: existing.key
          }
        }
      });

      return existing;
    });

    return {
      deleted: true,
      config: this.serializeConfigEntry(deleted)
    };
  }

  serializeConfigEntry(entry: ConfigEntry, revealSensitive = false) {
    return {
      id: entry.id,
      projectId: entry.projectId,
      environment: entry.environment,
      key: entry.key,
      value:
        entry.isSensitive && !revealSensitive ? MASKED_VALUE : entry.value,
      valueType: entry.valueType,
      isSensitive: entry.isSensitive,
      updatedBy: entry.updatedBy,
      createdAt: entry.createdAt,
      updatedAt: entry.updatedAt
    };
  }

  private isUniqueConstraintError(error: unknown) {
    return (
      error instanceof Prisma.PrismaClientKnownRequestError &&
      error.code === 'P2002'
    );
  }
}
