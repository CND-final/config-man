import { Controller, Get } from '@nestjs/common';
import { TemplatesService } from './templates.service';

@Controller('templates')
export class TemplatesController {
  constructor(private readonly templatesService: TemplatesService) {}

  @Get('base')
  getBaseTemplate() {
    return {
      name: 'Base Application Template',
      entries: this.templatesService.getBaseTemplateEntries()
    };
  }
}
