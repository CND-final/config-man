import {
  IsBoolean,
  IsEnum,
  IsOptional,
  IsString,
  MaxLength
} from 'class-validator';
import { ConfigValueType } from '@prisma/client';

export class UpdateConfigDto {
  @IsOptional()
  @IsString()
  value?: string;

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
