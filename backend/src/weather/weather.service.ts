import { Injectable } from '@nestjs/common';
import { InjectModel } from '@nestjs/mongoose';
import { Model } from 'mongoose';
import { CreateWeatherLogDto } from './dto/create-weather-log.dto';
import { WeatherLog } from './interfaces/weather-log.interface';
import { WeatherLog as WeatherLogModel, WeatherLogDocument } from './schemas/weather-log.schema';

@Injectable()
export class WeatherService {
  constructor(
    @InjectModel(WeatherLogModel.name) private weatherLogModel: Model<WeatherLogDocument>
  ) {}

  async createWeatherLog(createWeatherLogDto: CreateWeatherLogDto): Promise<WeatherLog> {
    const weatherLog = new this.weatherLogModel({
      ...createWeatherLogDto,
      timestamp: createWeatherLogDto.timestamp || new Date(),
    });

    const savedLog = await weatherLog.save();
    return {
      id: savedLog._id.toString(),
      temperatura: savedLog.temperatura,
      umidade: savedLog.umidade,
      vento: savedLog.vento,
      ceu: savedLog.ceu,
      probabilidade_chuva: savedLog.probabilidade_chuva,
      timestamp: savedLog.timestamp,
    };
  }

  async getAllWeatherLogs(): Promise<WeatherLog[]> {
    const logs = await this.weatherLogModel.find().exec();
    return logs.map(log => ({
      id: log._id.toString(),
      temperatura: log.temperatura,
      umidade: log.umidade,
      vento: log.vento,
      ceu: log.ceu,
      probabilidade_chuva: log.probabilidade_chuva,
      timestamp: log.timestamp,
    }));
  }

  async getWeatherLogById(id: string): Promise<WeatherLog | undefined> {
    const log = await this.weatherLogModel.findById(id).exec();
    if (!log) return undefined;

    return {
      id: log._id.toString(),
      temperatura: log.temperatura,
      umidade: log.umidade,
      vento: log.vento,
      ceu: log.ceu,
      probabilidade_chuva: log.probabilidade_chuva,
      timestamp: log.timestamp,
    };
  }
}
