import { Injectable } from '@nestjs/common';
import { ConfigValueType } from '@prisma/client';

export interface TemplateEntry {
  key: string;
  defaultValue: string;
  valueType: ConfigValueType;
  required: boolean;
  isSensitive: boolean;
  description: string;
}

@Injectable()
export class TemplatesService {
  private readonly baseTemplateEntries: TemplateEntry[] = [
    {
      key: 'app.timezone',
      defaultValue: 'Asia/Taipei',
      valueType: ConfigValueType.string,
      required: true,
      isSensitive: false,
      description: 'Default application timezone'
    },
    {
      key: 'log.level',
      defaultValue: 'info',
      valueType: ConfigValueType.string,
      required: true,
      isSensitive: false,
      description: 'Application logging level'
    },
    {
      key: 'api.baseUrl',
      defaultValue: 'https://api.example.com',
      valueType: ConfigValueType.string,
      required: true,
      isSensitive: false,
      description: 'Backend API base URL'
    },
    {
      key: 'feature.newCheckoutEnabled',
      defaultValue: 'false',
      valueType: ConfigValueType.boolean,
      required: false,
      isSensitive: false,
      description: 'Feature flag for staged rollout'
    },
    {
      key: 'database.url',
      defaultValue: '',
      valueType: ConfigValueType.string,
      required: true,
      isSensitive: true,
      description: 'Database connection string'
    }
  ];

  getBaseTemplateEntries() {
    return this.baseTemplateEntries;
  }
}
