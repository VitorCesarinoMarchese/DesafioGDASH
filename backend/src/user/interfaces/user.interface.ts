export interface User {
  id: string;
  email: string;
  password: string;
  refresh_token: string;
  creation_date: Date;
  last_update: Date;
}
