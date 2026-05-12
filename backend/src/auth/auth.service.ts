import { ForbiddenException, Injectable, UnauthorizedException } from '@nestjs/common';
import { DemoUser, UserRole } from './auth.types';

const DEMO_PASSWORD = 'password';

const DEMO_USERS: DemoUser[] = [
  {
    id: 'alice',
    email: 'admin@config-man.local',
    name: 'Alice Lin',
    role: 'system_admin'
  },
  {
    id: 'paul',
    email: 'project-admin@config-man.local',
    name: 'Paul Wu',
    role: 'project_admin'
  },
  {
    id: 'nora',
    email: 'developer@config-man.local',
    name: 'Nora Chen',
    role: 'developer'
  },
  {
    id: 'rachel',
    email: 'reviewer@config-man.local',
    name: 'Rachel Kao',
    role: 'reviewer'
  },
  {
    id: 'vincent',
    email: 'viewer@config-man.local',
    name: 'Vincent Lee',
    role: 'viewer'
  }
];

@Injectable()
export class AuthService {
  login(email: string, password: string) {
    const user = DEMO_USERS.find(
      (candidate) => candidate.email.toLowerCase() === email.toLowerCase()
    );

    if (!user || password !== DEMO_PASSWORD) {
      throw new UnauthorizedException('Invalid email or password');
    }

    return {
      token: user.id,
      user
    };
  }

  requireUser(authorization?: string) {
    const token = this.extractToken(authorization);
    const user = DEMO_USERS.find((candidate) => candidate.id === token);

    if (!user) {
      throw new UnauthorizedException('Missing or invalid login token');
    }

    return user;
  }

  requireAnyRole(user: DemoUser, roles: UserRole[]) {
    if (!roles.includes(user.role)) {
      throw new ForbiddenException(`Role "${user.role}" is not allowed`);
    }
  }

  canWriteEnvironment(user: DemoUser, environment: string) {
    if (['system_admin', 'project_admin'].includes(user.role)) {
      return true;
    }
    return user.role === 'developer' && environment !== 'prod';
  }

  requireConfigWrite(user: DemoUser, environment: string) {
    if (!this.canWriteEnvironment(user, environment)) {
      throw new ForbiddenException(
        `Role "${user.role}" cannot modify "${environment}" config`
      );
    }
  }

  requireReviewPermission(user: DemoUser) {
    this.requireAnyRole(user, ['system_admin', 'reviewer']);
  }

  requireReviewCreation(user: DemoUser) {
    this.requireAnyRole(user, [
      'system_admin',
      'project_admin',
      'developer',
      'reviewer'
    ]);
  }

  listDemoUsers() {
    return DEMO_USERS;
  }

  private extractToken(authorization?: string) {
    if (!authorization) {
      return '';
    }

    const [scheme, token] = authorization.split(' ');
    return scheme?.toLowerCase() === 'bearer' ? token : authorization;
  }
}
