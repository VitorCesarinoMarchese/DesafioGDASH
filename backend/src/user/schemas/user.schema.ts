import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { Document } from 'mongoose';

export type UserDocument = User & Document;

@Schema({ timestamps: true })
export class User {
  @Prop({ required: true, unique: true })
  email: string;

  @Prop({ required: true })
  password: string;

  @Prop()
  refresh_token: string;

  @Prop({ default: Date.now })
  creation_date: Date;

  @Prop({ default: Date.now })
  last_update: Date;

}

export const UserSchema = SchemaFactory.createForClass(User);
