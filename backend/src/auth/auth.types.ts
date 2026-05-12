export type UserRole =
  | 'system_admin'
  | 'project_admin'
  | 'developer'
  | 'reviewer'
  | 'viewer';

export interface DemoUser {
  id: string;
  email: string;
  name: string;
  role: UserRole;
}
