export interface WeatherLog {
  id: string;
  timestamp: Date;
  temperatura: number;
  umidade: number;
  vento: number;
  ceu: number;
  probabilidade_chuva: number;
}
