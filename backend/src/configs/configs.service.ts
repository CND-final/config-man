import {
  BadRequestException,
  ConflictException,
  ForbiddenException,
  Injectable,
  NotFoundException
} from '@nestjs/common';
import { ConfigEntry, ConfigValueType, Prisma } from '@prisma/client';
import { AuthService } from '../auth/auth.service';
import { DemoUser } from '../auth/auth.types';
import { PrismaService } from '../prisma/prisma.service';
import { ProjectsService } from '../projects/projects.service';
import { CreateConfigDto } from './dto/create-config.dto';
import { ImportConfigDto } from './dto/import-config.dto';
import { UpdateConfigDto } from './dto/update-config.dto';

const MASKED_VALUE = '******';

@Injectable()
export class ConfigsService {
  constructor(
    private readonly prisma: PrismaService,
    private readonly projectsService: ProjectsService,
    private readonly authService: AuthService
  ) {}

  async listConfigs(
    user: DemoUser,
    projectId: string,
    environment: string,
    revealSensitive = false
  ) {
    if (!environment) {
      throw new BadRequestException('Query parameter "env" is required');
    }

    await this.projectsService.ensureEnvironmentExists(projectId, environment);

    if (
      revealSensitive &&
      !['system_admin', 'project_admin', 'developer'].includes(user.role)
    ) {
      throw new ForbiddenException('Role cannot reveal sensitive values');
    }

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

  async createConfig(projectId: string, dto: CreateConfigDto, user: DemoUser) {
    this.authService.requireConfigWrite(user, dto.environment);
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
            updatedBy: user.name
          }
        });

        await tx.configVersion.create({
          data: {
            configId: created.id,
            oldValue: null,
            newValue: created.value,
            changedBy: user.name,
            changeReason: dto.changeReason ?? 'create config'
          }
        });

        await tx.auditLog.create({
          data: {
            actor: user.name,
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
    user: DemoUser
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

      this.authService.requireConfigWrite(user, existing.environment);

      const entry = await tx.configEntry.update({
        where: { id: configId },
        data: {
          value: dto.value ?? existing.value,
          valueType: dto.valueType ?? existing.valueType,
          isSensitive: dto.isSensitive ?? existing.isSensitive,
          updatedBy: user.name
        }
      });

      await tx.configVersion.create({
        data: {
          configId,
          oldValue: existing.value,
          newValue: entry.value,
          changedBy: user.name,
          changeReason: dto.changeReason ?? 'update config'
        }
      });

      await tx.auditLog.create({
        data: {
          actor: user.name,
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

  async deleteConfig(projectId: string, configId: string, user: DemoUser) {
    const deleted = await this.prisma.$transaction(async (tx) => {
      const existing = await tx.configEntry.findFirst({
        where: { id: configId, projectId }
      });

      if (!existing) {
        throw new NotFoundException(
          `Config "${configId}" not found for project "${projectId}"`
        );
      }

      this.authService.requireConfigWrite(user, existing.environment);

      await tx.configEntry.delete({ where: { id: configId } });

      await tx.auditLog.create({
        data: {
          actor: user.name,
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

  async importConfig(projectId: string, dto: ImportConfigDto, user: DemoUser) {
    this.authService.requireConfigWrite(user, dto.environment);
    await this.projectsService.ensureEnvironmentExists(projectId, dto.environment);

    const parsedEntries = this.parseConfigFile(dto.format, dto.content);
    if (parsedEntries.length === 0) {
      throw new BadRequestException('No config entries found in file content');
    }

    const result = await this.prisma.$transaction(async (tx) => {
      let created = 0;
      let updated = 0;
      let unchanged = 0;

      for (const parsed of parsedEntries) {
        const existing = await tx.configEntry.findUnique({
          where: {
            projectId_environment_key: {
              projectId,
              environment: dto.environment,
              key: parsed.key
            }
          }
        });

        if (!existing) {
          const entry = await tx.configEntry.create({
            data: {
              projectId,
              environment: dto.environment,
              key: parsed.key,
              value: parsed.value,
              valueType: parsed.valueType,
              isSensitive: this.looksSensitive(parsed.key),
              updatedBy: user.name
            }
          });

          await tx.configVersion.create({
            data: {
              configId: entry.id,
              oldValue: null,
              newValue: parsed.value,
              changedBy: user.name,
              changeReason: dto.changeReason
            }
          });
          created += 1;
          continue;
        }

        if (
          existing.value === parsed.value &&
          existing.valueType === parsed.valueType
        ) {
          unchanged += 1;
          continue;
        }

        await tx.configEntry.update({
          where: { id: existing.id },
          data: {
            value: parsed.value,
            valueType: parsed.valueType,
            isSensitive: existing.isSensitive || this.looksSensitive(parsed.key),
            updatedBy: user.name
          }
        });

        await tx.configVersion.create({
          data: {
            configId: existing.id,
            oldValue: existing.value,
            newValue: parsed.value,
            changedBy: user.name,
            changeReason: dto.changeReason
          }
        });
        updated += 1;
      }

      await tx.auditLog.create({
        data: {
          actor: user.name,
          action: 'config.import',
          resourceType: 'config_file',
          projectId,
          metadata: {
            environment: dto.environment,
            format: dto.format,
            created,
            updated,
            unchanged
          }
        }
      });

      return { created, updated, unchanged };
    });

    return {
      projectId,
      environment: dto.environment,
      format: dto.format,
      imported: parsedEntries.length,
      ...result
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

  private parseConfigFile(
    format: ImportConfigDto['format'],
    content: string
  ): Array<{ key: string; value: string; valueType: ConfigValueType }> {
    if (format === 'json') {
      const parsed = JSON.parse(content) as unknown;
      return this.flattenObject(parsed).map(([key, value]) => ({
        key,
        value: this.stringifyValue(value),
        valueType: this.inferValueType(value)
      }));
    }

    if (format === 'properties') {
      return content
        .split(/\r?\n/)
        .map((line) => line.trim())
        .filter((line) => line && !line.startsWith('#') && !line.startsWith('!'))
        .map((line) => {
          const separator = line.includes('=') ? '=' : ':';
          const [rawKey, ...valueParts] = line.split(separator);
          const value = valueParts.join(separator).trim();
          return {
            key: rawKey.trim(),
            value,
            valueType: this.inferValueType(value)
          };
        })
        .filter((entry) => entry.key);
    }

    return this.parseSimpleYaml(content).map(([key, value]) => ({
      key,
      value,
      valueType: this.inferValueType(value)
    }));
  }

  private flattenObject(value: unknown, prefix = ''): Array<[string, unknown]> {
    if (value === null || typeof value !== 'object' || Array.isArray(value)) {
      return prefix ? [[prefix, value]] : [];
    }

    return Object.entries(value).flatMap(([key, nestedValue]) => {
      const nextPrefix = prefix ? `${prefix}.${key}` : key;
      return this.flattenObject(nestedValue, nextPrefix);
    });
  }

  private parseSimpleYaml(content: string): Array<[string, string]> {
    const entries: Array<[string, string]> = [];
    const stack: Array<{ indent: number; path: string[] }> = [
      { indent: -1, path: [] }
    ];

    for (const rawLine of content.split(/\r?\n/)) {
      if (!rawLine.trim() || rawLine.trim().startsWith('#')) {
        continue;
      }

      const match = rawLine.match(/^(\s*)([^:#]+):\s*(.*)$/);
      if (!match) {
        continue;
      }

      const indent = match[1].length;
      const key = match[2].trim();
      const rawValue = match[3].trim();

      while (stack.length > 1 && indent <= stack[stack.length - 1].indent) {
        stack.pop();
      }

      const path = [...stack[stack.length - 1].path, key];
      if (!rawValue) {
        stack.push({ indent, path });
        continue;
      }

      entries.push([path.join('.'), rawValue.replace(/^['"]|['"]$/g, '')]);
    }

    return entries;
  }

  private stringifyValue(value: unknown) {
    if (typeof value === 'string') {
      return value;
    }
    if (value === null || value === undefined) {
      return '';
    }
    return typeof value === 'object' ? JSON.stringify(value) : String(value);
  }

  private inferValueType(value: unknown): ConfigValueType {
    if (typeof value === 'boolean') {
      return ConfigValueType.boolean;
    }
    if (typeof value === 'number') {
      return ConfigValueType.number;
    }
    if (typeof value === 'object' && value !== null) {
      return ConfigValueType.json;
    }

    const stringValue = String(value).trim();
    if (['true', 'false'].includes(stringValue.toLowerCase())) {
      return ConfigValueType.boolean;
    }
    if (stringValue !== '' && Number.isFinite(Number(stringValue))) {
      return ConfigValueType.number;
    }
    if (
      (stringValue.startsWith('{') && stringValue.endsWith('}')) ||
      (stringValue.startsWith('[') && stringValue.endsWith(']'))
    ) {
      return ConfigValueType.json;
    }
    return ConfigValueType.string;
  }

  private looksSensitive(key: string) {
    return /(password|secret|token|credential|database\.url|db\.url)/i.test(key);
  }
}
