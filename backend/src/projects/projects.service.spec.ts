import { ProjectsService } from './projects.service';

describe('ProjectsService', () => {
  it('creates default dev/staging/prod environments when registering a project', async () => {
    const project = {
      id: 'project-1',
      name: 'checkout',
      description: null,
      repoUrl: null,
      ownerName: 'Platform Team',
      defaultFormat: 'yaml',
      createdAt: new Date('2026-05-12T00:00:00.000Z'),
      updatedAt: new Date('2026-05-12T00:00:00.000Z'),
      environments: [
        { id: 'env-1', name: 'dev', sortOrder: 1 },
        { id: 'env-2', name: 'staging', sortOrder: 2 },
        { id: 'env-3', name: 'prod', sortOrder: 3 }
      ],
      _count: { configEntries: 0 }
    };

    const tx = {
      project: {
        create: jest.fn().mockResolvedValue({ id: 'project-1', name: 'checkout' }),
        findUniqueOrThrow: jest.fn().mockResolvedValue(project)
      },
      projectEnvironment: { createMany: jest.fn() },
      auditLog: { create: jest.fn() }
    };
    const prisma = {
      $transaction: jest.fn((callback) => callback(tx))
    } as any;
    const service = new ProjectsService(prisma);

    const result = await service.createProject(
      { name: 'checkout', ownerName: 'Platform Team' },
      'alice'
    );

    expect(tx.projectEnvironment.createMany).toHaveBeenCalledWith({
      data: [
        { projectId: 'project-1', name: 'dev', sortOrder: 1 },
        { projectId: 'project-1', name: 'staging', sortOrder: 2 },
        { projectId: 'project-1', name: 'prod', sortOrder: 3 }
      ]
    });
    expect(tx.auditLog.create).toHaveBeenCalledWith(
      expect.objectContaining({
        data: expect.objectContaining({
          actor: 'alice',
          action: 'project.create',
          resourceId: 'project-1'
        })
      })
    );
    expect(result.environments.map((environment) => environment.name)).toEqual([
      'dev',
      'staging',
      'prod'
    ]);
  });
});
