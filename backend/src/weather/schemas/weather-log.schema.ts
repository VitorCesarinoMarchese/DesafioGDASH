import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { Document } from 'mongoose';

export type WeatherLogDocument = WeatherLog & Document;

@Schema({ timestamps: true })
export class WeatherLog {
  @Prop({ required: true })
  temperatura: number;

  @Prop({ required: true })
  umidade: number;

  @Prop({ required: true })
  vento: number;

  @Prop({ required: true })
  ceu: number;

  @Prop({ required: true })
  probabilidade_chuva: number;

  @Prop({ default: Date.now })
  timestamp: Date;
}

export const WeatherLogSchema = SchemaFactory.createForClass(WeatherLog);
