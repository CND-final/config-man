import {
  IsBoolean,
  IsEnum,
  IsOptional,
  IsString,
  MaxLength,
  MinLength
} from 'class-validator';
import { ConfigValueType } from '@prisma/client';

export class CreateConfigDto {
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

  @IsOptional()
  @IsString()
  @MaxLength(500)
  changeReason?: string;
}
