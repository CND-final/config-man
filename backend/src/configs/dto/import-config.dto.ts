import { IsIn, IsString, MaxLength, MinLength } from 'class-validator';

export class ImportConfigDto {
  @IsString()
  @MinLength(1)
  @MaxLength(60)
  environment: string;

  @IsIn(['json', 'yaml', 'properties'])
  format: 'json' | 'yaml' | 'properties';

  @IsString()
  @MinLength(1)
  content: string;

  @IsString()
  @MaxLength(500)
  changeReason = 'import config file';
}
