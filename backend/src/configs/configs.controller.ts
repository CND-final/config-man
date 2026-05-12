import {
  Body,
  Controller,
  Delete,
  Get,
  Headers,
  Param,
  Post,
  Put,
  Query
} from '@nestjs/common';
import { AuthService } from '../auth/auth.service';
import { CreateConfigDto } from './dto/create-config.dto';
import { ImportConfigDto } from './dto/import-config.dto';
import { UpdateConfigDto } from './dto/update-config.dto';
import { ConfigsService } from './configs.service';

@Controller('projects/:projectId/configs')
export class ConfigsController {
  constructor(
    private readonly authService: AuthService,
    private readonly configsService: ConfigsService
  ) {}

  @Get()
  listConfigs(
    @Param('projectId') projectId: string,
    @Query('env') environment: string,
    @Query('revealSensitive') revealSensitive?: string,
    @Headers('authorization') authorization?: string
  ) {
    const user = this.authService.requireUser(authorization);
    return this.configsService.listConfigs(
      user,
      projectId,
      environment,
      revealSensitive === 'true'
    );
  }

  @Post()
  createConfig(
    @Param('projectId') projectId: string,
    @Body() dto: CreateConfigDto,
    @Headers('authorization') authorization?: string
  ) {
    const user = this.authService.requireUser(authorization);
    return this.configsService.createConfig(projectId, dto, user);
  }

  @Post('import')
  importConfig(
    @Param('projectId') projectId: string,
    @Body() dto: ImportConfigDto,
    @Headers('authorization') authorization?: string
  ) {
    const user = this.authService.requireUser(authorization);
    return this.configsService.importConfig(projectId, dto, user);
  }

  @Put(':configId')
  updateConfig(
    @Param('projectId') projectId: string,
    @Param('configId') configId: string,
    @Body() dto: UpdateConfigDto,
    @Headers('authorization') authorization?: string
  ) {
    const user = this.authService.requireUser(authorization);
    return this.configsService.updateConfig(projectId, configId, dto, user);
  }

  @Delete(':configId')
  deleteConfig(
    @Param('projectId') projectId: string,
    @Param('configId') configId: string,
    @Headers('authorization') authorization?: string
  ) {
    const user = this.authService.requireUser(authorization);
    return this.configsService.deleteConfig(projectId, configId, user);
  }
}
