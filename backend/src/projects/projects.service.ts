import {
  ConflictException,
  Injectable,
  NotFoundException
} from '@nestjs/common';
import { Prisma, Project, ProjectEnvironment } from '@prisma/client';
import { PrismaService } from '../prisma/prisma.service';
import { CreateProjectDto } from './dto/create-project.dto';

const DEFAULT_ENVIRONMENTS = ['dev', 'staging', 'prod'];

type ProjectWithEnvironments = Project & {
  environments: ProjectEnvironment[];
};

@Injectable()
export class ProjectsService {
  constructor(private readonly prisma: PrismaService) {}

  async listProjects() {
    const projects = await this.prisma.project.findMany({
      orderBy: { createdAt: 'desc' },
      include: {
        environments: { orderBy: { sortOrder: 'asc' } },
        _count: { select: { configEntries: true } }
      }
    });

    return projects.map((project) => this.serializeProject(project));
  }

  async createProject(dto: CreateProjectDto, actor: string) {
    const environments = this.normalizeEnvironments(dto.environments);

    try {
      const project = await this.prisma.$transaction(async (tx) => {
        const createdProject = await tx.project.create({
          data: {
            name: dto.name,
            description: dto.description,
            repoUrl: dto.repoUrl,
            ownerName: dto.ownerName,
            defaultFormat: dto.defaultFormat ?? 'yaml'
          }
        });

        await tx.projectEnvironment.createMany({
          data: environments.map((name, index) => ({
            projectId: createdProject.id,
            name,
            sortOrder: index + 1
          }))
        });

        await tx.auditLog.create({
          data: {
            actor,
            action: 'project.create',
            resourceType: 'project',
            resourceId: createdProject.id,
            projectId: createdProject.id,
            metadata: { name: createdProject.name, environments }
          }
        });

        return tx.project.findUniqueOrThrow({
          where: { id: createdProject.id },
          include: {
            environments: { orderBy: { sortOrder: 'asc' } },
            _count: { select: { configEntries: true } }
          }
        });
      });

      return this.serializeProject(project);
    } catch (error) {
      if (this.isUniqueConstraintError(error)) {
        throw new ConflictException(`Project "${dto.name}" already exists`);
      }
      throw error;
    }
  }

  async getProject(projectId: string) {
    const project = await this.prisma.project.findUnique({
      where: { id: projectId },
      include: {
        environments: { orderBy: { sortOrder: 'asc' } },
        _count: { select: { configEntries: true } }
      }
    });

    if (!project) {
      throw new NotFoundException(`Project "${projectId}" not found`);
    }

    return this.serializeProject(project);
  }

  async ensureProjectExists(projectId: string) {
    const project = await this.prisma.project.findUnique({
      where: { id: projectId },
      include: { environments: true }
    });

    if (!project) {
      throw new NotFoundException(`Project "${projectId}" not found`);
    }

    return project;
  }

  async ensureEnvironmentExists(projectId: string, environment: string) {
    await this.ensureProjectExists(projectId);

    const projectEnvironment = await this.prisma.projectEnvironment.findUnique({
      where: { projectId_name: { projectId, name: environment } }
    });

    if (!projectEnvironment) {
      throw new NotFoundException(
        `Environment "${environment}" not found for project "${projectId}"`
      );
    }

    return projectEnvironment;
  }

  private normalizeEnvironments(environments?: string[]) {
    const names = environments?.length ? environments : DEFAULT_ENVIRONMENTS;
    const normalized = names
      .map((name) => name.trim().toLowerCase())
      .filter(Boolean);

    return [...new Set(normalized)];
  }

  private serializeProject(
    project: ProjectWithEnvironments & {
      _count?: { configEntries: number };
    }
  ) {
    return {
      id: project.id,
      name: project.name,
      description: project.description,
      repoUrl: project.repoUrl,
      ownerName: project.ownerName,
      defaultFormat: project.defaultFormat,
      environments: project.environments.map((environment) => ({
        id: environment.id,
        name: environment.name,
        sortOrder: environment.sortOrder
      })),
      configCount: project._count?.configEntries ?? 0,
      createdAt: project.createdAt,
      updatedAt: project.updatedAt
    };
  }

  private isUniqueConstraintError(error: unknown) {
    return (
      error instanceof Prisma.PrismaClientKnownRequestError &&
      error.code === 'P2002'
    );
  }
}
