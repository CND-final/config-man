import { ConflictException } from '@nestjs/common';
import { ConfigValueType, Prisma } from '@prisma/client';
import { ConfigsService } from './configs.service';

describe('ConfigsService', () => {
  const projectsService = {
    ensureEnvironmentExists: jest.fn()
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('masks sensitive config values by default', async () => {
    const prisma = {
      configEntry: {
        findMany: jest.fn().mockResolvedValue([
          {
            id: 'config-1',
            projectId: 'project-1',
            environment: 'prod',
            key: 'database.url',
            value: 'postgresql://secret',
            valueType: ConfigValueType.string,
            isSensitive: true,
            updatedBy: 'alice',
            createdAt: new Date('2026-05-12T00:00:00.000Z'),
            updatedAt: new Date('2026-05-12T00:00:00.000Z')
          }
        ])
      }
    } as any;
    const service = new ConfigsService(prisma, projectsService as any);

    const result = await service.listConfigs('project-1', 'prod');

    expect(result.entries[0].value).toBe('******');
  });

  it('returns conflict when the same key already exists in the same project environment', async () => {
    const prismaError = Object.assign(new Error('unique constraint'), {
      code: 'P2002'
    });
    Object.setPrototypeOf(
      prismaError,
      Prisma.PrismaClientKnownRequestError.prototype
    );

    const prisma = {
      $transaction: jest.fn().mockRejectedValue(prismaError)
    } as any;
    const service = new ConfigsService(prisma, projectsService as any);

    await expect(
      service.createConfig(
        'project-1',
        {
          environment: 'dev',
          key: 'api.baseUrl',
          value: 'https://dev-api.example.com'
        },
        'alice'
      )
    ).rejects.toBeInstanceOf(ConflictException);
  });

  it('writes a version record when updating config', async () => {
    const existing = {
      id: 'config-1',
      projectId: 'project-1',
      environment: 'dev',
      key: 'log.level',
      value: 'info',
      valueType: ConfigValueType.string,
      isSensitive: false,
      updatedBy: 'alice',
      createdAt: new Date('2026-05-12T00:00:00.000Z'),
      updatedAt: new Date('2026-05-12T00:00:00.000Z')
    };
    const updated = { ...existing, value: 'debug', updatedBy: 'bob' };
    const tx = {
      configEntry: {
        findFirst: jest.fn().mockResolvedValue(existing),
        update: jest.fn().mockResolvedValue(updated)
      },
      configVersion: { create: jest.fn() },
      auditLog: { create: jest.fn() }
    };
    const prisma = {
      $transaction: jest.fn((callback) => callback(tx))
    } as any;
    const service = new ConfigsService(prisma, projectsService as any);

    await service.updateConfig(
      'project-1',
      'config-1',
      { value: 'debug', changeReason: 'raise log level for debugging' },
      'bob'
    );

    expect(tx.configVersion.create).toHaveBeenCalledWith({
      data: {
        configId: 'config-1',
        oldValue: 'info',
        newValue: 'debug',
        changedBy: 'bob',
        changeReason: 'raise log level for debugging'
      }
    });
  });
});
