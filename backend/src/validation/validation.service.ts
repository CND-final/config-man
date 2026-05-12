import { Injectable } from '@nestjs/common';
import { ConfigEntry, ConfigValueType } from '@prisma/client';
import { PrismaService } from '../prisma/prisma.service';
import { ProjectsService } from '../projects/projects.service';
import { TemplatesService } from '../templates/templates.service';
import { ValidateProjectDto } from './validate-project.dto';

export interface ValidationIssue {
  environment: string;
  key: string;
  code: string;
  message: string;
}

export interface ValidationResult {
  projectId: string;
  environments: string[];
  valid: boolean;
  errors: ValidationIssue[];
  warnings: ValidationIssue[];
}

type ValidationEntry = Pick<
  ConfigEntry,
  'environment' | 'key' | 'value' | 'valueType' | 'isSensitive'
>;

@Injectable()
export class ValidationService {
  constructor(
    private readonly prisma: PrismaService,
    private readonly projectsService: ProjectsService,
    private readonly templatesService: TemplatesService
  ) {}

  async validateProject(
    projectId: string,
    dto: ValidateProjectDto = {}
  ): Promise<ValidationResult> {
    const project = await this.projectsService.ensureProjectExists(projectId);
    const targetEnvironments = dto.environment
      ? [dto.environment]
      : project.environments
          .sort((a, b) => a.sortOrder - b.sortOrder)
          .map((environment) => environment.name);

    if (dto.environment) {
      await this.projectsService.ensureEnvironmentExists(
        projectId,
        dto.environment
      );
    }

    const persistedEntries = await this.prisma.configEntry.findMany({
      where: {
        projectId,
        environment: { in: targetEnvironments }
      }
    });

    const draftEntries = (dto.draftEntries ?? []).filter((entry) =>
      targetEnvironments.includes(entry.environment)
    );

    const allEntries: ValidationEntry[] = [
      ...persistedEntries,
      ...draftEntries.map((entry) => ({
        environment: entry.environment,
        key: entry.key,
        value: entry.value,
        valueType: entry.valueType ?? ConfigValueType.string,
        isSensitive: entry.isSensitive ?? false
      }))
    ];

    const errors: ValidationIssue[] = [];
    const warnings: ValidationIssue[] = [];
    const requiredTemplateEntries = this.templatesService
      .getBaseTemplateEntries()
      .filter((entry) => entry.required);

    for (const environment of targetEnvironments) {
      const entriesForEnvironment = allEntries.filter(
        (entry) => entry.environment === environment
      );
      const keys = new Set(entriesForEnvironment.map((entry) => entry.key));

      for (const templateEntry of requiredTemplateEntries) {
        if (!keys.has(templateEntry.key)) {
          errors.push({
            environment,
            key: templateEntry.key,
            code: 'missing_required_key',
            message: `Missing required config key "${templateEntry.key}"`
          });
        }
      }

      errors.push(
        ...this.findDuplicateKeyIssues(environment, entriesForEnvironment)
      );

      for (const entry of entriesForEnvironment) {
        if (!this.valueMatchesType(entry.value, entry.valueType)) {
          errors.push({
            environment,
            key: entry.key,
            code: 'invalid_value_type',
            message: `Value does not match type "${entry.valueType}"`
          });
        }

        if (this.looksSensitive(entry.key) && !entry.isSensitive) {
          warnings.push({
            environment,
            key: entry.key,
            code: 'sensitive_key_not_marked',
            message: `Key "${entry.key}" looks sensitive but is not marked sensitive`
          });
        }
      }
    }

    return {
      projectId,
      environments: targetEnvironments,
      valid: errors.length === 0,
      errors,
      warnings
    };
  }

  private findDuplicateKeyIssues(
    environment: string,
    entries: ValidationEntry[]
  ): ValidationIssue[] {
    const counts = new Map<string, number>();

    for (const entry of entries) {
      counts.set(entry.key, (counts.get(entry.key) ?? 0) + 1);
    }

    return [...counts.entries()]
      .filter(([, count]) => count > 1)
      .map(([key]) => ({
        environment,
        key,
        code: 'duplicate_key',
        message: `Duplicate config key "${key}" in "${environment}"`
      }));
  }

  private valueMatchesType(value: string, valueType: ConfigValueType) {
    switch (valueType) {
      case ConfigValueType.number:
        return value.trim() !== '' && Number.isFinite(Number(value));
      case ConfigValueType.boolean:
        return ['true', 'false'].includes(value.toLowerCase());
      case ConfigValueType.json:
        try {
          JSON.parse(value);
          return true;
        } catch {
          return false;
        }
      case ConfigValueType.string:
      default:
        return true;
    }
  }

  private looksSensitive(key: string) {
    return /(password|secret|token|credential|database\.url)/i.test(key);
  }
}
