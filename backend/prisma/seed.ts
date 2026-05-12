import { PrismaClient, ConfigValueType } from '@prisma/client';

const prisma = new PrismaClient();

async function main() {
  const project = await prisma.project.upsert({
    where: { name: 'customer-portal' },
    update: {},
    create: {
      name: 'customer-portal',
      description: 'Demo project for config-man phase 1',
      repoUrl: 'https://git.example.com/platform/customer-portal',
      ownerName: 'Platform Team',
      defaultFormat: 'yaml',
      environments: {
        create: [
          { name: 'dev', sortOrder: 1 },
          { name: 'staging', sortOrder: 2 },
          { name: 'prod', sortOrder: 3 }
        ]
      }
    }
  });

  const entries = [
    {
      environment: 'dev',
      key: 'api.baseUrl',
      value: 'https://dev-api.example.com',
      valueType: ConfigValueType.string
    },
    {
      environment: 'dev',
      key: 'log.level',
      value: 'debug',
      valueType: ConfigValueType.string
    },
    {
      environment: 'prod',
      key: 'api.baseUrl',
      value: 'https://api.example.com',
      valueType: ConfigValueType.string
    },
    {
      environment: 'prod',
      key: 'database.url',
      value: 'postgresql://prod-user:secret@prod-db:5432/app',
      valueType: ConfigValueType.string,
      isSensitive: true
    }
  ];

  for (const entry of entries) {
    const config = await prisma.configEntry.upsert({
      where: {
        projectId_environment_key: {
          projectId: project.id,
          environment: entry.environment,
          key: entry.key
        }
      },
      update: {},
      create: {
        projectId: project.id,
        environment: entry.environment,
        key: entry.key,
        value: entry.value,
        valueType: entry.valueType,
        isSensitive: entry.isSensitive ?? false,
        updatedBy: 'seed-admin'
      }
    });

    await prisma.configVersion.upsert({
      where: { id: `${config.id}-seed` },
      update: {},
      create: {
        id: `${config.id}-seed`,
        configId: config.id,
        oldValue: null,
        newValue: entry.value,
        changedBy: 'seed-admin',
        changeReason: 'seed demo config'
      }
    });
  }

  await prisma.auditLog.create({
    data: {
      actor: 'seed-admin',
      action: 'seed_demo_project',
      resourceType: 'project',
      resourceId: project.id,
      projectId: project.id,
      metadata: { projectName: project.name }
    }
  });
}

main()
  .catch((error) => {
    console.error(error);
    process.exit(1);
  })
  .finally(async () => {
    await prisma.$disconnect();
  });
