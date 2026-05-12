import { IsOptional, IsString, MaxLength, MinLength } from 'class-validator';

export class CreateReviewRequestDto {
  @IsString()
  @MinLength(1)
  projectId: string;

  @IsString()
  @MinLength(1)
  @MaxLength(60)
  environment: string;

  @IsOptional()
  @IsString()
  @MaxLength(240)
  configKey?: string;

  @IsString()
  @MinLength(4)
  @MaxLength(500)
  reason: string;
}
