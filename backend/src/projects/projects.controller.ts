import { Body, Controller, Get, Headers, Param, Post } from '@nestjs/common';
import { CreateProjectDto } from './dto/create-project.dto';
import { ProjectsService } from './projects.service';

@Controller('projects')
export class ProjectsController {
  constructor(private readonly projectsService: ProjectsService) {}

  @Get()
  listProjects() {
    return this.projectsService.listProjects();
  }

  @Post()
  createProject(
    @Body() dto: CreateProjectDto,
    @Headers('x-actor') actor = 'seed-admin'
  ) {
    return this.projectsService.createProject(dto, actor);
  }

  @Get(':projectId')
  getProject(@Param('projectId') projectId: string) {
    return this.projectsService.getProject(projectId);
  }
}
