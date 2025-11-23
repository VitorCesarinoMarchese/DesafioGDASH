import { Controller, Post, Body, Get, Param } from '@nestjs/common';
import { WeatherService } from './weather.service';
import { CreateWeatherLogDto } from './dto/create-weather-log.dto';
import type { WeatherLog } from './interfaces/weather-log.interface';

@Controller('api/weather')
export class WeatherController {
  constructor(private readonly weatherService: WeatherService) { }

  @Post('logs')
  async createWeatherLog(
    @Body() createWeatherLogDto: CreateWeatherLogDto,
  ): Promise<WeatherLog> {
    return this.weatherService.createWeatherLog(createWeatherLogDto);
  }

  @Get('logs')
  async getAllWeatherLogs(): Promise<WeatherLog[]> {
    return this.weatherService.getAllWeatherLogs();
  }

  @Get('logs/:id')
  async getWeatherLogById(@Param('id') id: string): Promise<WeatherLog | undefined> {
    return this.weatherService.getWeatherLogById(id);
  }
}

