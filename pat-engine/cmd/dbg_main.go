package main
import (
 "fmt"
 "pat-engine/internal/broker"
 "pat-engine/internal/config"
 "pat-engine/internal/signal"
 "pat-engine/internal/strategy"
 "pat-engine/internal/types"
)
func main(){
 s:=&types.MarketState{
  Symbol:"XAUUSD",Timeframe:types.TFM1,CurrentPrice:2000.00,Spread:0.20,ATR:1.00,
  Indicators:types.Indicators{EMA9:2001.0,EMA21:2000.5,EMA50:1999.0,EMA100:1990.0,EMA200:1980.0,SMA200:1980.0,ADX:30,ADXPlusDI:25,ADXMinusDI:10,RSI:55,MACDMain:0.5,MACDSignal:0.3,OsMA:0.2,StochMain:0.6,StochSignal:0.4,BollUpper:2010,BollLower:1990},
  MTFScore:15,Regime:"TRENDING_BULLISH",Session:types.Session{CurrentSession:"LONDON"},Quality:"AUTHORITATIVE",
  Candle:types.Candle{IsBullish:true,IsDisplacement:true},
 }
 st:=strategy.Must("ULTRA_SCALPING")
 cfg:=config.DefaultUltraScalping()
 pol:=&broker.BrokerPolicy{Symbol:"XAUUSD",AllowsScalping:true,Digits:2}
 d:=signal.Decide(s,st,cfg,pol)
 fmt.Printf("exec=%v dir=%s score=%.1f reasons=%v\n",d.Signal.Executable,d.Signal.Direction,d.Signal.RawScore,d.Reasons)
}
