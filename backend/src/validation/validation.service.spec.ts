import { ConfigValueType } from '@prisma/client';
import { ValidationService } from './validation.service';

describe('ValidationService', () => {
  const project = {
    id: 'project-1',
    environments: [{ name: 'dev', sortOrder: 1 }]
  };
  const projectsService = {
    ensureProjectExists: jest.fn().mockResolvedValue(project),
    ensureEnvironmentExists: jest.fn()
  };
  const templatesService = {
    getBaseTemplateEntries: jest.fn().mockReturnValue([
      {
        key: 'api.baseUrl',
        required: true,
        valueType: ConfigValueType.string
      },
      {
        key: 'database.url',
        required: true,
        valueType: ConfigValueType.string
      }
    ])
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('reports missing required keys', async () => {
    const prisma = {
      configEntry: {
        findMany: jest.fn().mockResolvedValue([
          {
            environment: 'dev',
            key: 'api.baseUrl',
            value: 'https://dev-api.example.com',
            valueType: ConfigValueType.string,
            isSensitive: false
          }
        ])
      }
    } as any;
    const service = new ValidationService(
      prisma,
      projectsService as any,
      templatesService as any
    );

    const result = await service.validateProject('project-1', {
      environment: 'dev'
    });

    expect(result.valid).toBe(false);
    expect(result.errors).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          code: 'missing_required_key',
          key: 'database.url'
        })
      ])
    );
  });

  it('reports duplicate keys in draft entries before import', async () => {
    const prisma = {
      configEntry: {
        findMany: jest.fn().mockResolvedValue([])
      }
    } as any;
    const service = new ValidationService(
      prisma,
      projectsService as any,
      templatesService as any
    );

    const result = await service.validateProject('project-1', {
      environment: 'dev',
      draftEntries: [
        {
          environment: 'dev',
          key: 'api.baseUrl',
          value: 'https://one.example.com',
          valueType: ConfigValueType.string
        },
        {
          environment: 'dev',
          key: 'api.baseUrl',
          value: 'https://two.example.com',
          valueType: ConfigValueType.string
        }
      ]
    });

    expect(result.errors).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          code: 'duplicate_key',
          key: 'api.baseUrl'
        })
      ])
    );
  });
});
