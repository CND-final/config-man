import { Body, Controller, Param, Post } from '@nestjs/common';
import { ValidateProjectDto } from './validate-project.dto';
import { ValidationService } from './validation.service';

@Controller('projects/:projectId/validate')
export class ValidationController {
  constructor(private readonly validationService: ValidationService) {}

  @Post()
  validateProject(
    @Param('projectId') projectId: string,
    @Body() dto: ValidateProjectDto
  ) {
    return this.validationService.validateProject(projectId, dto);
  }
}
