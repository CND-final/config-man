import { Type } from 'class-transformer';
import {
  IsArray,
  IsBoolean,
  IsEnum,
  IsOptional,
  IsString,
  MaxLength,
  MinLength,
  ValidateNested
} from 'class-validator';
import { ConfigValueType } from '@prisma/client';

export class DraftConfigEntryDto {
  @IsString()
  @MinLength(1)
  @MaxLength(60)
  environment: string;

  @IsString()
  @MinLength(1)
  @MaxLength(240)
  key: string;

  @IsString()
  value: string;

  @IsOptional()
  @IsEnum(ConfigValueType)
  valueType?: ConfigValueType;

  @IsOptional()
  @IsBoolean()
  isSensitive?: boolean;
}

export class ValidateProjectDto {
  @IsOptional()
  @IsString()
  @MaxLength(60)
  environment?: string;

  @IsOptional()
  @IsArray()
  @ValidateNested({ each: true })
  @Type(() => DraftConfigEntryDto)
  draftEntries?: DraftConfigEntryDto[];
}
