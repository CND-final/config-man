import { Body, Controller, Get, Headers, Param, Post } from '@nestjs/common';
import { AuthService } from '../auth/auth.service';
import { CreateProjectDto } from './dto/create-project.dto';
import { ProjectsService } from './projects.service';

@Controller('projects')
export class ProjectsController {
  constructor(
    private readonly authService: AuthService,
    private readonly projectsService: ProjectsService
  ) {}

  @Get()
  listProjects(@Headers('authorization') authorization?: string) {
    this.authService.requireUser(authorization);
    return this.projectsService.listProjects();
  }

  @Post()
  createProject(
    @Body() dto: CreateProjectDto,
    @Headers('authorization') authorization?: string
  ) {
    const user = this.authService.requireUser(authorization);
    this.authService.requireAnyRole(user, ['system_admin', 'project_admin']);
    return this.projectsService.createProject(dto, user.name);
  }

  @Get(':projectId')
  getProject(
    @Param('projectId') projectId: string,
    @Headers('authorization') authorization?: string
  ) {
    this.authService.requireUser(authorization);
    return this.projectsService.getProject(projectId);
  }
}
