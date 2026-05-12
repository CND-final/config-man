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
import { CreateConfigDto } from './dto/create-config.dto';
import { UpdateConfigDto } from './dto/update-config.dto';
import { ConfigsService } from './configs.service';

@Controller('projects/:projectId/configs')
export class ConfigsController {
  constructor(private readonly configsService: ConfigsService) {}

  @Get()
  listConfigs(
    @Param('projectId') projectId: string,
    @Query('env') environment: string,
    @Query('revealSensitive') revealSensitive?: string
  ) {
    return this.configsService.listConfigs(
      projectId,
      environment,
      revealSensitive === 'true'
    );
  }

  @Post()
  createConfig(
    @Param('projectId') projectId: string,
    @Body() dto: CreateConfigDto,
    @Headers('x-actor') actor = 'seed-admin'
  ) {
    return this.configsService.createConfig(projectId, dto, actor);
  }

  @Put(':configId')
  updateConfig(
    @Param('projectId') projectId: string,
    @Param('configId') configId: string,
    @Body() dto: UpdateConfigDto,
    @Headers('x-actor') actor = 'seed-admin'
  ) {
    return this.configsService.updateConfig(projectId, configId, dto, actor);
  }

  @Delete(':configId')
  deleteConfig(
    @Param('projectId') projectId: string,
    @Param('configId') configId: string,
    @Headers('x-actor') actor = 'seed-admin'
  ) {
    return this.configsService.deleteConfig(projectId, configId, actor);
  }
}
