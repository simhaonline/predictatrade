//+------------------------------------------------------------------+
//|                                          Predict-A-Trade v1.2.mq5|
//|                        Ultra Scalping - Production Grade         |
//|                        https://predictatrade.com                |
//+------------------------------------------------------------------+
#property version   "1.2"
#property description "Predict-A-Trade v1.2 | Production ready ultra scalping"
#property description "Website: https://predictatrade.com"
#property description "Account auto-detection, real cost model, equity protection"

#include <Trade\Trade.mqh>
CTrade trade;

//====================== INPUTS =====================================
input group "== Symbol Profile (auto if empty) =="
input string InpSymbolProfile   = "";    // e.g., "XAUUSD", "XAGUSD", "EURUSD", etc.

input group "== Identity & Safety =="
input long   InpMagic             = 980100;
input int    InpSlippagePoints    = 0;           // 0 = dynamic slippage based on spread
input double InpRiskPct           = 0.5;         // intended risk per trade
input double InpMaxRiskPct        = 2.0;         // absolute maximum risk per trade
input double InpCostPips          = 0.0;         // 0 = auto from spread+commission
input int    InpMaxTradesPerDay   = 50;
input double InpDailyLossCapPct   = 3.0;
input bool   InpUseNewsFilter     = true;        // turn ON for production
input bool   InpTradeOnlyOverlap  = false;
input int    InpMaxBarsInTrade    = 12;          // time stop (bars)
input int    InpMaxSecondsInTrade = 120;         // time stop (seconds)

input group "== Entry Quality =="
input int    InpMinScore          = 40;
input int    InpScoreMargin       = 2;
input double InpSL_ATR            = 0.6;
input double InpMinSL_Pips        = 0.0;         // auto
input double InpMaxSL_Pips        = 0.0;         // auto

input group "== Trade Management =="
input double InpTP1_R             = 0.9;
input double InpTP2_R             = 1.4;
input double InpTP3_R             = 2.0;
input double InpTP1_Pct           = 100;
input double InpTP2_Pct           = 0;
input double InpBE_BufferPips     = 1.0;
input bool   InpUseTrail          = true;
input double InpTrailStart_R      = 0.35;
input double InpTrailStep_ATR     = 0.25;

input group "== Dashboard =="
input int    InpPanelX            = 12;
input int    InpPanelY            = 24;
input bool   InpShowAccount       = true;

//====================== ENUMS / STRUCTS ============================
enum ENUM_REGIME { REGIME_NONE, REGIME_TRENDING_BULLISH, REGIME_TRENDING_BEARISH,
                   REGIME_RANGE, REGIME_MEAN_REVERSION, REGIME_BREAKOUT, REGIME_HIGH_VOLATILITY };
enum ENUM_SESSION { SESS_OFF_HOURS, SESS_SYDNEY, SESS_TOKYO, SESS_LONDON, SESS_NEW_YORK, SESS_OVERLAP };
enum ENUM_STTREND { ST_NONE, ST_BULL, ST_BEAR, ST_RANGE };

struct SHandles {
   int ema9,ema21,ema50,ema100,ema200, sma50,sma100,sma200;
   int macd,adx,sar,ichi,rsi,stoch,cci,atr,bb;
   int mtf[4];
   int atrH1;
};
struct SFeatures {
   double ema9,ema21,ema50,ema100,ema200,sma50,sma100,sma200;
   int    emaCross921;
   double macdMain,macdSig,macdHist; int macdBullX,macdBearX;
   double adx,plusDI,minusDI;
   double sar; int sarLong;
   double tenkan,kijun,senkouA,senkouB,chikou,cloudTop,cloudBot; int ichimokuPos;
   double rsi,stochK,stochD,srsiRaw,srsiK,srsiD,cci;
   double atr,bbUp,bbMid,bbLow,bbWidth,bbwPct,bbwZ,atrZ;
   int    bbBullRev,bbBearRev;
   double obv,obvZ,tickVolZ,vwap,vwapUp,vwapLow,rvwapN;
   ENUM_STTREND structTrend;
   int bosBull,bosBear,chochBull,chochBear,mss;
   double lastSwingHigh,lastSwingLow;
   double liqPoolHigh[3],liqPoolLow[3]; int poolHighTouches[3],poolLowTouches[3];
   int sweepHigh,sweepLow;
   double fvgBullTop,fvgBullBot,fvgBearTop,fvgBearBot;
   double obBullTop,obBullBot,obBearTop,obBearBot; int breakerActive;
   double fibLevel[6]; double fibExt[4]; int inGoldenZone; double confluenceScore;
   double r150Hi,r150Lo,r150Mid;
   double pivD_P,pivD_R1,pivD_R2,pivD_R3,pivD_S1,pivD_S2,pivD_S3;
   double pivW_P,pivW_R1,pivW_R2,pivW_R3,pivW_S1,pivW_S2,pivW_S3;
   ENUM_REGIME regime; double mtfScore; string mtfStates;
   ENUM_SESSION session; int isOverlap,isWeekend; double newsRisk;
   double bodySize,upWick,loWick,bodyRatio,atrNormRange;
   int patDoji,patPinBull,patPinBear,patEngulfBull,patEngulfBear,patInside,patOutside;
   int displBull,displBear,rejection,compression,expansion,breakoutCandle,consecBull,consecBear;
   double avgATR;
   double sessionHigh, sessionLow;
   double tickDeltaZ;
};
struct STrade {
   bool active; ulong ticket; int dir;
   double entry,rDist,sl,tp1,tp2,tp3,lots0; datetime openTime;
   bool beDone,tp1Done,tp2Done;
   double riskMoney;
};
struct SAccInfo {
   long   leverage;
   int    digits;
   double point;
   double spreadPts,spreadPips,spreadCostPerLot;
   double tickValue,tickSize,pointValuePerLot,pipValuePerLot;
   double contractSize;
   long   stopsLevel,freezeLevel;
   double swapLong,swapShort;
   double volMin,volMax,volStep;
   double balance,equity,marginFree,marginUsed,marginLevel;
   string currency;
   string accountType;
   double commissionPerLot;
};

//====================== GLOBALS ====================================
SHandles H; SFeatures F; STrade T; SAccInfo A;
#define LOOKBACK 400
#define SWING_K  2
#define SCORE_MAX 136.0
#define PFX "PAT_"

double  g_rawB,g_rawS; int g_scoreB,g_scoreS,g_bestDir,g_bestScore;
string  g_decision="init";
int     g_dayTrades,g_wins,g_losses,g_scratch;
double  g_dayPL,g_floating,g_Rmultiple;
string  g_drv="-";
uint    g_lastPanel=0;
datetime g_lastBar=0;

string g_profileName;
double g_pipSize;
double g_minSLPrice, g_maxSLPrice;
double g_tp1Price, g_tp2Price, g_tp3Price;
int    g_spreadLimit;
double g_scoreThreshold;

double g_dayStartBalance = 0;
datetime g_statsDay = 0;

//====================== SMALL UTILS ================================
double Buf(int handle,int buffer,int shift){
   double v[1];
   if(CopyBuffer(handle,buffer,shift,1,v)!=1) return EMPTY_VALUE;
   return v[0];
}
double Pip(){
   int d=(int)SymbolInfoInteger(_Symbol,SYMBOL_DIGITS);
   return (d==3||d==5)?10.0*_Point:_Point;
}
double MinStopDist(){
   long s=SymbolInfoInteger(_Symbol,SYMBOL_TRADE_STOPS_LEVEL);
   long f=SymbolInfoInteger(_Symbol,SYMBOL_TRADE_FREEZE_LEVEL);
   long m=(s>f)?s:f;
   return (double)m*_Point;
}
double NormLots(double lots){
   double mn=SymbolInfoDouble(_Symbol,SYMBOL_VOLUME_MIN);
   double mx=SymbolInfoDouble(_Symbol,SYMBOL_VOLUME_MAX);
   double st=SymbolInfoDouble(_Symbol,SYMBOL_VOLUME_STEP);
   if(st<=0) st=0.01;
   lots=MathFloor(lots/st)*st;
   return MathMax(mn,MathMin(mx,lots));
}
bool IsNewBar(){
   datetime t=iTime(_Symbol,_Period,0);
   if(t!=g_lastBar){ g_lastBar=t; return true; }
   return false;
}
ulong FindMyTicket(){
   for(int i=PositionsTotal()-1;i>=0;i--){
      ulong tk=PositionGetTicket(i);
      if(tk==0) continue;
      if(PositionGetString(POSITION_SYMBOL)==_Symbol &&
         PositionGetInteger(POSITION_MAGIC)==InpMagic) return tk;
   }
   return 0;
}
void Reject(string reason){
   g_decision = reason;
   PrintFormat("PAT REJECT | %s | B=%d S=%d | spread=%d/%d | ATR=%.5f",
               reason, g_scoreB, g_scoreS,
               (int)SymbolInfoInteger(_Symbol,SYMBOL_SPREAD),
               g_spreadLimit, F.atr);
}

//====================== ACCOUNT / BROKER INFO ======================
void UpdateAccountInfo(){
   A.leverage     = AccountInfoInteger(ACCOUNT_LEVERAGE);
   A.digits       = (int)SymbolInfoInteger(_Symbol,SYMBOL_DIGITS);
   A.point        = SymbolInfoDouble(_Symbol,SYMBOL_POINT);
   A.contractSize = SymbolInfoDouble(_Symbol,SYMBOL_TRADE_CONTRACT_SIZE);
   A.tickValue    = SymbolInfoDouble(_Symbol,SYMBOL_TRADE_TICK_VALUE);
   A.tickSize     = SymbolInfoDouble(_Symbol,SYMBOL_TRADE_TICK_SIZE);
   A.spreadPts    = (SymbolInfoDouble(_Symbol,SYMBOL_ASK) - SymbolInfoDouble(_Symbol,SYMBOL_BID))/A.point;
   double pip     = Pip();
   A.spreadPips   = A.spreadPts*A.point/pip;
   A.pointValuePerLot = (A.tickSize>0)? A.tickValue*(A.point/A.tickSize) : 0;
   A.pipValuePerLot   = (A.tickSize>0)? A.tickValue*(pip  /A.tickSize) : 0;
   A.spreadCostPerLot = A.spreadPts*A.pointValuePerLot;
   A.stopsLevel   = SymbolInfoInteger(_Symbol,SYMBOL_TRADE_STOPS_LEVEL);
   A.freezeLevel  = SymbolInfoInteger(_Symbol,SYMBOL_TRADE_FREEZE_LEVEL);
   A.swapLong     = SymbolInfoDouble(_Symbol,SYMBOL_SWAP_LONG);
   A.swapShort    = SymbolInfoDouble(_Symbol,SYMBOL_SWAP_SHORT);
   A.volMin       = SymbolInfoDouble(_Symbol,SYMBOL_VOLUME_MIN);
   A.volMax       = SymbolInfoDouble(_Symbol,SYMBOL_VOLUME_MAX);
   A.volStep      = SymbolInfoDouble(_Symbol,SYMBOL_VOLUME_STEP);
   A.balance      = AccountInfoDouble(ACCOUNT_BALANCE);
   A.equity       = AccountInfoDouble(ACCOUNT_EQUITY);
   A.marginFree   = AccountInfoDouble(ACCOUNT_MARGIN_FREE);
   A.marginUsed   = MathMax(0.0, A.equity - A.marginFree);
   A.marginLevel  = AccountInfoDouble(ACCOUNT_MARGIN_LEVEL);
   if(A.marginLevel<=0 && A.marginUsed>0) A.marginLevel=A.equity/A.marginUsed*100.0;
   A.currency     = AccountInfoString(ACCOUNT_CURRENCY);
   // Commission
   A.commissionPerLot = SymbolInfoDouble(_Symbol, SYMBOL_TRADE_COMMISSION);
   if(A.commissionPerLot == 0) A.commissionPerLot = 7.0; // fallback
   // Account type detection
   ENUM_ACCOUNT_TRADE_MODE mode = (ENUM_ACCOUNT_TRADE_MODE)AccountInfoInteger(ACCOUNT_TRADE_MODE);
   string modeStr = (mode == ACCOUNT_TRADE_MODE_DEMO) ? "DEMO" :
                    (mode == ACCOUNT_TRADE_MODE_CONTEST) ? "CONTEST" : "REAL";
   bool isCent = (StringFind(_Symbol, "cent") >= 0 || StringFind(A.currency, "C") == StringLen(A.currency)-1);
   bool isIslamic = (A.swapLong == 0 && A.swapShort == 0);
   bool isECN = (A.spreadPts < 5 || (AccountInfoInteger(ACCOUNT_TRADE_EXECUTION) == ACCOUNT_TRADE_EXECUTION_MARKET));
   A.accountType = StringFormat("%s | %s%s%s", modeStr,
                                isCent ? "CENT " : "",
                                isECN ? "ECN/RAW " : "STANDARD ",
                                isIslamic ? "ISLAMIC" : "");
}

//====================== SYMBOL PROFILE SETUP =======================
void SetSymbolProfile(){
   string sym = (InpSymbolProfile != "") ? InpSymbolProfile : _Symbol;
   g_profileName = sym;
   
   if(StringFind(sym,"XAU")>=0 || StringFind(sym,"GOLD")>=0){
      g_pipSize = 0.01;
      g_minSLPrice = 0.30;
      g_maxSLPrice = 1.50;
      g_spreadLimit = 80;
      g_scoreThreshold = 40;
   }
   else if(StringFind(sym,"XAG")>=0 || StringFind(sym,"SILVER")>=0){
      g_pipSize = 0.001;
      g_minSLPrice = 0.08;
      g_maxSLPrice = 0.40;
      g_spreadLimit = 60;
      g_scoreThreshold = 40;
   }
   else if(StringFind(sym,"EUR")>=0 || StringFind(sym,"GBP")>=0 || StringFind(sym,"USD")>=0){
      g_pipSize = Pip();
      g_minSLPrice = 8 * g_pipSize;
      g_maxSLPrice = 12 * g_pipSize;
      g_spreadLimit = 20;
      g_scoreThreshold = 40;
   }
   else {
      g_pipSize = Pip();
      g_minSLPrice = 8 * g_pipSize;
      g_maxSLPrice = 12 * g_pipSize;
      g_spreadLimit = 20;
      g_scoreThreshold = 40;
   }
   if(InpMinSL_Pips > 0) g_minSLPrice = InpMinSL_Pips * g_pipSize;
   if(InpMaxSL_Pips > 0) g_maxSLPrice = InpMaxSL_Pips * g_pipSize;
   if(InpMinScore > 0) g_scoreThreshold = InpMinScore;
   // Dynamic spread limit: current spread + 50% buffer, never below base
   double spreadBuffer = A.spreadPts * 1.5;
   if(spreadBuffer > g_spreadLimit) g_spreadLimit = (int)MathCeil(spreadBuffer);
}

//====================== STATE PERSIST ==============================
string SVKey(string k){ return PFX+_Symbol+"_"+(string)InpMagic+"_"+k; }
void SaveState(){
   GlobalVariableSet(SVKey("act"),T.active?1:0);
   GlobalVariableSet(SVKey("dir"),T.dir);
   GlobalVariableSet(SVKey("entry"),T.entry);
   GlobalVariableSet(SVKey("r"),T.rDist);
   GlobalVariableSet(SVKey("tp1"),T.tp1);
   GlobalVariableSet(SVKey("tp2"),T.tp2);
   GlobalVariableSet(SVKey("tp3"),T.tp3);
   GlobalVariableSet(SVKey("lots"),T.lots0);
   GlobalVariableSet(SVKey("ot"),(double)T.openTime);
   GlobalVariableSet(SVKey("be"),T.beDone?1:0);
   GlobalVariableSet(SVKey("p1"),T.tp1Done?1:0);
   GlobalVariableSet(SVKey("p2"),T.tp2Done?1:0);
   GlobalVariableSet(SVKey("risk"),T.riskMoney);
   GlobalVariableSet(SVKey("day_start_balance"),g_dayStartBalance);
   GlobalVariableSet(SVKey("day_stats_date"),(double)g_statsDay);
}
void LoadState(){
   if(!GlobalVariableCheck(SVKey("act"))) return;
   T.active  = GlobalVariableGet(SVKey("act"))>0.5;
   T.dir     = (int)GlobalVariableGet(SVKey("dir"));
   T.entry   = GlobalVariableGet(SVKey("entry"));
   T.rDist   = GlobalVariableGet(SVKey("r"));
   T.tp1     = GlobalVariableGet(SVKey("tp1"));
   T.tp2     = GlobalVariableGet(SVKey("tp2"));
   T.tp3     = GlobalVariableGet(SVKey("tp3"));
   T.lots0   = GlobalVariableGet(SVKey("lots"));
   T.openTime= (datetime)GlobalVariableGet(SVKey("ot"));
   T.beDone  = GlobalVariableGet(SVKey("be"))>0.5;
   T.tp1Done = GlobalVariableGet(SVKey("p1"))>0.5;
   T.tp2Done = GlobalVariableGet(SVKey("p2"))>0.5;
   T.riskMoney = GlobalVariableGet(SVKey("risk"));
   if(GlobalVariableCheck(SVKey("day_start_balance")))
      g_dayStartBalance = GlobalVariableGet(SVKey("day_start_balance"));
   if(GlobalVariableCheck(SVKey("day_stats_date")))
      g_statsDay = (datetime)GlobalVariableGet(SVKey("day_stats_date"));
}

//====================== HANDLE INDICATORS ==========================
bool FeaturesInit(){
   H.ema9  =iMA(_Symbol,_Period,9,  0,MODE_EMA,PRICE_CLOSE);
   H.ema21 =iMA(_Symbol,_Period,21, 0,MODE_EMA,PRICE_CLOSE);
   H.ema50 =iMA(_Symbol,_Period,50, 0,MODE_EMA,PRICE_CLOSE);
   H.ema100=iMA(_Symbol,_Period,100,0,MODE_EMA,PRICE_CLOSE);
   H.ema200=iMA(_Symbol,_Period,200,0,MODE_EMA,PRICE_CLOSE);
   H.sma50 =iMA(_Symbol,_Period,50, 0,MODE_SMA,PRICE_CLOSE);
   H.sma100=iMA(_Symbol,_Period,100,0,MODE_SMA,PRICE_CLOSE);
   H.sma200=iMA(_Symbol,_Period,200,0,MODE_SMA,PRICE_CLOSE);
   H.macd  =iMACD(_Symbol,_Period,12,26,9,PRICE_CLOSE);
   H.adx   =iADX(_Symbol,_Period,14);
   H.sar   =iSAR(_Symbol,_Period,0.02,0.2);
   H.ichi  =iIchimoku(_Symbol,_Period,9,26,52);
   H.rsi   =iRSI(_Symbol,_Period,14,PRICE_CLOSE);
   H.stoch =iStochastic(_Symbol,_Period,14,3,3,MODE_SMA,STO_LOWHIGH);
   H.cci   =iCCI(_Symbol,_Period,14,PRICE_TYPICAL);
   H.atr   =iATR(_Symbol,_Period,14);
   H.bb    =iBands(_Symbol,_Period,20,0,2.0,PRICE_CLOSE);
   H.atrH1 =iATR(_Symbol,PERIOD_H1,14);
   ENUM_TIMEFRAMES mp[4]={PERIOD_M5,PERIOD_M15,PERIOD_H1,PERIOD_H4};
   for(int i=0;i<4;i++) H.mtf[i]=iMA(_Symbol,mp[i],50,0,MODE_EMA,PRICE_CLOSE);
   int hs[]={H.ema9,H.ema21,H.ema50,H.ema100,H.ema200,H.sma50,H.sma100,H.sma200,
             H.macd,H.adx,H.sar,H.ichi,H.rsi,H.stoch,H.cci,H.atr,H.bb,H.atrH1};
   for(int i=0;i<ArraySize(hs);i++) if(hs[i]==INVALID_HANDLE) return false;
   return true;
}

void UpdateHandleIndicators(){
   F.ema9=Buf(H.ema9,0,1);  F.ema21=Buf(H.ema21,0,1);  F.ema50=Buf(H.ema50,0,1);
   F.ema100=Buf(H.ema100,0,1); F.ema200=Buf(H.ema200,0,1);
   F.sma50=Buf(H.sma50,0,1); F.sma100=Buf(H.sma100,0,1); F.sma200=Buf(H.sma200,0,1);
   double e9p=Buf(H.ema9,0,2), e21p=Buf(H.ema21,0,2);
   F.emaCross921=(e9p<=e21p&&F.ema9>F.ema21)?1:(e9p>=e21p&&F.ema9<F.ema21)?-1:0;
   F.macdMain=Buf(H.macd,0,1); F.macdSig=Buf(H.macd,1,1); F.macdHist=F.macdMain-F.macdSig;
   F.macdBullX=(Buf(H.macd,0,2)<=Buf(H.macd,1,2)&&F.macdMain>F.macdSig)?1:0;
   F.macdBearX=(Buf(H.macd,0,2)>=Buf(H.macd,1,2)&&F.macdMain<F.macdSig)?1:0;
   F.adx=Buf(H.adx,0,1); F.plusDI=Buf(H.adx,1,1); F.minusDI=Buf(H.adx,2,1);
   F.sar=Buf(H.sar,0,1);
   F.sarLong=(F.sar!=EMPTY_VALUE && F.sar<iClose(_Symbol,_Period,1))?1:0;
   F.tenkan=Buf(H.ichi,0,1); F.kijun=Buf(H.ichi,1,1);
   F.senkouA=Buf(H.ichi,2,1); F.senkouB=Buf(H.ichi,3,1); F.chikou=Buf(H.ichi,4,1);
   if(F.senkouA!=EMPTY_VALUE&&F.senkouB!=EMPTY_VALUE){
      F.cloudTop=MathMax(F.senkouA,F.senkouB); F.cloudBot=MathMin(F.senkouA,F.senkouB);
      double c1=iClose(_Symbol,_Period,1);
      F.ichimokuPos=(c1>F.cloudTop)?1:(c1<F.cloudBot)?-1:0;
   }
   F.rsi=Buf(H.rsi,0,1); F.stochK=Buf(H.stoch,0,1); F.stochD=Buf(H.stoch,1,1);
   F.cci=Buf(H.cci,0,1);
   F.atr=Buf(H.atr,0,1);
   F.avgATR=Buf(H.atrH1,0,1);
   F.bbMid=Buf(H.bb,0,1); F.bbUp=Buf(H.bb,1,1); F.bbLow=Buf(H.bb,2,1);
   F.bbWidth=(F.bbMid>0&&F.bbMid!=EMPTY_VALUE)?(F.bbUp-F.bbLow)/F.bbMid:0;
}

//====================== STOCH RSI & SESSION / VOLUME DELTA =========
double SRSIRawAt(const double &rsi[],int shift,int n){
   int mi=ArrayMinimum(rsi,shift,n), ma=ArrayMaximum(rsi,shift,n);
   if(mi<0||ma<0) return 50;
   double mn=rsi[mi],mx=rsi[ma];
   return (mx-mn>0)?100.0*(rsi[shift]-mn)/(mx-mn):50;
}
void UpdateStochRSI(){
   int n=14; double r[]; ArraySetAsSeries(r,true);
   if(CopyBuffer(H.rsi,0,0,n+8,r)<n+8) return;
   double w1=SRSIRawAt(r,1,n),w2=SRSIRawAt(r,2,n),w3=SRSIRawAt(r,3,n),
          w4=SRSIRawAt(r,4,n),w5=SRSIRawAt(r,5,n);
   F.srsiRaw=w1;
   F.srsiK=(w1+w2+w3)/3.0;
   F.srsiD=(F.srsiK+(w2+w3+w4)/3.0+(w3+w4+w5)/3.0)/3.0;
}
void UpdateSessionHighLowAndVolumeDelta(){
   MqlRates r[]; ArraySetAsSeries(r,true);
   if(CopyRates(_Symbol,_Period,0,200,r)<200) return;
   datetime dayStart=iTime(_Symbol,PERIOD_D1,0);
   int barCount = iBarShift(_Symbol,_Period,dayStart,false);
   if(barCount < 0) barCount = 200;
   barCount = MathMin(barCount, ArraySize(r)-1);
   if(barCount < 1) return;

   double hi = -DBL_MAX, lo = DBL_MAX;
   for(int i=1; i<barCount; i++){
      if(r[i].high > hi) hi = r[i].high;
      if(r[i].low  < lo) lo = r[i].low;
   }
   F.sessionHigh = hi;
   F.sessionLow  = lo;

   double upVol=0, downVol=0;
   for(int i=1; i<=20; i++){
      if(r[i].close > r[i].open) upVol += (double)r[i].tick_volume;
      else if(r[i].close < r[i].open) downVol += (double)r[i].tick_volume;
   }
   double delta = upVol - downVol;
   double mean=0, sd=0;
   for(int i=1; i<=50; i++){
      double d = (r[i].close > r[i].open) ? (double)r[i].tick_volume : 
                (r[i].close < r[i].open) ? -(double)r[i].tick_volume : 0;
      mean += d; sd += d*d;
   }
   mean /= 50; sd = MathSqrt(sd/50 - mean*mean);
   F.tickDeltaZ = (sd > 0) ? delta / sd : 0;
}

//====================== VOLATILITY EXTRAS ==========================
void UpdateVolatilityExtras(){
   double up[],lo[],mid[],at[];
   ArraySetAsSeries(up,true);ArraySetAsSeries(lo,true);
   ArraySetAsSeries(mid,true);ArraySetAsSeries(at,true);
   int N=200;
   bool ok = CopyBuffer(H.bb,1,0,N,up)==N
          && CopyBuffer(H.bb,2,0,N,lo)==N
          && CopyBuffer(H.bb,0,0,N,mid)==N
          && CopyBuffer(H.atr,0,0,101,at)==101;
   if(!ok) return;
   double w[]; ArrayResize(w,N);
   for(int i=0;i<N;i++) w[i]=(mid[i]>0)?(up[i]-lo[i])/mid[i]:0;
   int below=0; for(int i=1;i<N;i++) if(w[i]<w[1]) below++;
   F.bbwPct=100.0*below/(N-1);
   double m=0,s=0;
   for(int i=1;i<=100;i++) m+=w[i]; m/=100.0;
   for(int i=1;i<=100;i++) s+=(w[i]-m)*(w[i]-m);
   double sd=MathSqrt(s/100.0);
   F.bbwZ=(sd>0)?(w[1]-m)/sd:0;
   double c2=iClose(_Symbol,_Period,2),c1=iClose(_Symbol,_Period,1);
   F.bbBearRev=(c2>up[1]&&c1<up[1])?1:0;
   F.bbBullRev=(c2<lo[1]&&c1>lo[1])?1:0;
   m=0;s=0;
   for(int i=1;i<=100;i++) m+=at[i]; m/=100.0;
   for(int i=1;i<=100;i++) s+=(at[i]-m)*(at[i]-m);
   sd=MathSqrt(s/100.0);
   F.atrZ=(sd>0)?(at[1]-m)/sd:0;
}

//====================== VOLUME / VWAP ==============================
void UpdateVolumeFeatures(){
   MqlRates r[]; ArraySetAsSeries(r,true);
   if(CopyRates(_Symbol,_Period,0,LOOKBACK,r)<LOOKBACK) return;
   double obv[]; ArrayResize(obv,LOOKBACK);
   obv[LOOKBACK-1]=0;
   for(int i=LOOKBACK-2;i>=1;i--)
      obv[i]=obv[i+1]+(r[i].close>r[i+1].close?(double)r[i].tick_volume:
             (r[i].close<r[i+1].close?-(double)r[i].tick_volume:0));
   F.obv=obv[1];
   int W=100; double m=0,s=0;
   for(int i=1;i<=W;i++) m+=obv[i]; m/=W;
   for(int i=1;i<=W;i++) s+=(obv[i]-m)*(obv[i]-m);
   s=MathSqrt(s/W); F.obvZ=(s>0)?(obv[1]-m)/s:0;
   m=0;s=0;
   for(int i=2;i<=W+1;i++) m+=(double)r[i].tick_volume; m/=W;
   for(int i=2;i<=W+1;i++) s+=((double)r[i].tick_volume-m)*((double)r[i].tick_volume-m);
   s=MathSqrt(s/W); F.tickVolZ=(s>0)?((double)r[1].tick_volume-m)/s:0;
   datetime dayStart=iTime(_Symbol,PERIOD_D1,0);
   MqlRates d[]; ArraySetAsSeries(d,true);
   int nd=CopyRates(_Symbol,_Period,dayStart,TimeCurrent(),d);
   if(nd>2000) nd=2000;
   if(nd>0){
      double pv=0,vv=0;
      for(int i=0;i<nd;i++){ double tp=(d[i].high+d[i].low+d[i].close)/3.0; pv+=tp*d[i].tick_volume; vv+=(double)d[i].tick_volume; }
      F.vwap=(vv>0)?pv/vv:iClose(_Symbol,_Period,1);
      double var=0;
      for(int i=0;i<nd;i++){ double tp=(d[i].high+d[i].low+d[i].close)/3.0; var+=(double)d[i].tick_volume*(tp-F.vwap)*(tp-F.vwap); }
      double sdv=MathSqrt(vv>0?var/vv:0);
      F.vwapUp=F.vwap+2.0*sdv; F.vwapLow=F.vwap-2.0*sdv;
   }
   double pv2=0,vv2=0;
   for(int i=1;i<=50;i++){ double tp=(r[i].high+r[i].low+r[i].close)/3.0; pv2+=tp*r[i].tick_volume; vv2+=(double)r[i].tick_volume; }
   F.rvwapN=(vv2>0)?pv2/vv2:0;
}

//====================== STRUCTURE / LIQUIDITY ======================
bool IsSwingHigh(const MqlRates &r[],int i,int total){
   if(i<SWING_K||i+SWING_K>=total) return false;
   for(int j=1;j<=SWING_K;j++)
      if(r[i].high<=r[i-j].high||r[i].high<=r[i+j].high) return false;
   return true;
}
bool IsSwingLow(const MqlRates &r[],int i,int total){
   if(i<SWING_K||i+SWING_K>=total) return false;
   for(int j=1;j<=SWING_K;j++)
      if(r[i].low>=r[i-j].low||r[i].low>=r[i+j].low) return false;
   return true;
}
void UpdateStructure(){
   MqlRates r[]; ArraySetAsSeries(r,true);
   if(CopyRates(_Symbol,_Period,0,LOOKBACK,r)<LOOKBACK) return;
   double atr=(F.atr>0&&F.atr!=EMPTY_VALUE)?F.atr:100*_Point;
   double sh[20],sl[20]; int shI[20],slI[20],nsh=0,nsl=0;
   for(int i=1;i<LOOKBACK-SWING_K&&(nsh<20||nsl<20);i++){
      if(nsh<20&&IsSwingHigh(r,i,LOOKBACK)){ sh[nsh]=r[i].high; shI[nsh]=i; nsh++; }
      if(nsl<20&&IsSwingLow (r,i,LOOKBACK)){ sl[nsl]=r[i].low;  slI[nsl]=i; nsl++; }
   }
   F.lastSwingHigh=(nsh>0)?sh[0]:0; F.lastSwingLow=(nsl>0)?sl[0]:0;
   if(nsh>=2&&nsl>=2)
      F.structTrend=(sh[0]>sh[1]&&sl[0]>sl[1])?ST_BULL:
                    (sh[0]<sh[1]&&sl[0]<sl[1])?ST_BEAR:ST_RANGE;
   else F.structTrend=ST_NONE;
   F.bosBull=F.bosBear=F.chochBull=F.chochBear=F.mss=0;
   double body1=MathAbs(r[1].close-r[1].open);
   for(int b=1;b<=2;b++){
      if(nsh>0&&r[b].close>sh[0]) F.bosBull=1;
      if(nsl>0&&r[b].close<sl[0]) F.bosBear=1;
      if(F.structTrend==ST_BULL&&nsl>0&&r[b].close<sl[0]) F.chochBear=1;
      if(F.structTrend==ST_BEAR&&nsh>0&&r[b].close>sh[0]) F.chochBull=1;
   }
   F.mss=((F.chochBull||F.chochBear)&&body1>1.5*atr)?1:0;
   double tol=0.15*atr;
   for(int k=0;k<3;k++){ F.liqPoolHigh[k]=0;F.liqPoolLow[k]=0;F.poolHighTouches[k]=0;F.poolLowTouches[k]=0; }
   bool usedH[20],usedL[20];
   for(int i=0;i<20;i++){ usedH[i]=false; usedL[i]=false; }
   int ph=0,pl=0;
   for(int i=0;i<nsh&&ph<3;i++){
      if(usedH[i]) continue; int t=0;
      for(int j=i+1;j<nsh;j++) if(!usedH[j]&&MathAbs(sh[i]-sh[j])<=tol){t++;usedH[j]=true;}
      if(t>=1){ F.liqPoolHigh[ph]=sh[i]; F.poolHighTouches[ph]=t+2; usedH[i]=true; ph++; }
   }
   for(int i=0;i<nsl&&pl<3;i++){
      if(usedL[i]) continue; int t=0;
      for(int j=i+1;j<nsl;j++) if(!usedL[j]&&MathAbs(sl[i]-sl[j])<=tol){t++;usedL[j]=true;}
      if(t>=1){ F.liqPoolLow[pl]=sl[i]; F.poolLowTouches[pl]=t+2; usedL[i]=true; pl++; }
   }
   F.sweepHigh=F.sweepLow=0;
   for(int b=1;b<=3;b++)
      for(int k=0;k<3;k++){
         if(F.liqPoolHigh[k]>0&&r[b].high>F.liqPoolHigh[k]&&r[b].close<F.liqPoolHigh[k]) F.sweepHigh=1;
         if(F.liqPoolLow[k]>0 &&r[b].low <F.liqPoolLow[k] &&r[b].close>F.liqPoolLow[k])  F.sweepLow=1;
      }
   F.fvgBullTop=F.fvgBullBot=F.fvgBearTop=F.fvgBearBot=0;
   for(int i=1;i<58;i++){
      if(r[i].low>r[i+2].high){
         bool filled=false;
         for(int j=0;j<i;j++) if(r[j].low<=r[i+2].high){filled=true;break;}
         if(!filled){F.fvgBullTop=r[i].low;F.fvgBullBot=r[i+2].high;break;}
      }
   }
   for(int i=1;i<58;i++){
      if(r[i].high<r[i+2].low){
         bool filled=false;
         for(int j=0;j<i;j++) if(r[j].high>=r[i+2].low){filled=true;break;}
         if(!filled){F.fvgBearTop=r[i+2].low;F.fvgBearBot=r[i].high;break;}
      }
   }
   F.obBullTop=F.obBullBot=F.obBearTop=F.obBearBot=0; F.breakerActive=0;
   if(F.bosBull==1&&nsh>0)
      for(int i=1;i<shI[0]+10&&i<LOOKBACK-2;i++)
         if(r[i].close<r[i].open){
            F.obBullTop=r[i].high; F.obBullBot=r[i].low;
            for(int j=0;j<i;j++) if(r[j].close<F.obBullBot) F.breakerActive=1;
            break;
         }
   if(F.bosBear==1&&nsl>0)
      for(int i=1;i<slI[0]+10&&i<LOOKBACK-2;i++)
         if(r[i].close>r[i].open){
            F.obBearTop=r[i].high; F.obBearBot=r[i].low;
            for(int j=0;j<i;j++) if(r[j].close>F.obBearBot) F.breakerActive=1;
            break;
         }
}

//====================== FIB / PIVOTS / CONFLUENCE ==================
void UpdateFibAndPivots(){
   MqlRates r[]; ArraySetAsSeries(r,true);
   if(CopyRates(_Symbol,_Period,0,LOOKBACK,r)<LOOKBACK) return;
   double atr=(F.atr>0&&F.atr!=EMPTY_VALUE)?F.atr:100*_Point;
   double L=DBL_MAX,Hh=-DBL_MAX; int li=0,hi=0;
   for(int i=1;i<150;i++){
      if(r[i].low<L){L=r[i].low;li=i;}
      if(r[i].high>Hh){Hh=r[i].high;hi=i;}
   }
   F.r150Hi=Hh; F.r150Lo=L; F.r150Mid=(Hh+L)*0.5;
   bool upLeg=(li>hi);
   double range=Hh-L; if(range<=0) return;
   double mult[6]={0.236,0.382,0.500,0.618,0.705,0.786};
   for(int k=0;k<6;k++)
      F.fibLevel[k]= upLeg? Hh-mult[k]*range : L+mult[k]*range;
   double extm[4]={1.272,1.618,2.000,2.618};
   for(int k=0;k<4;k++)
      F.fibExt[k]= upLeg? L+extm[k]*range : Hh-extm[k]*range;
   double gzA= upLeg? Hh-0.618*range : L+0.618*range;
   double gzB= upLeg? Hh-0.786*range : L+0.786*range;
   double c1=r[1].close;
   F.inGoldenZone=(c1>=MathMin(gzA,gzB)-2*_Point&&c1<=MathMax(gzA,gzB)+2*_Point)?1:0;

   MqlRates p[]; ArraySetAsSeries(p,true);
   if(CopyRates(_Symbol,PERIOD_D1,0,2,p)==2){
      double P=(p[1].high+p[1].low+p[1].close)/3.0, rng=p[1].high-p[1].low;
      F.pivD_P=P; F.pivD_R1=2*P-p[1].low; F.pivD_S1=2*P-p[1].high;
      F.pivD_R2=P+rng; F.pivD_S2=P-rng;
      F.pivD_R3=p[1].high+2*(P-p[1].low); F.pivD_S3=p[1].low-2*(p[1].high-P);
   }
   if(CopyRates(_Symbol,PERIOD_W1,0,2,p)==2){
      double P=(p[1].high+p[1].low+p[1].close)/3.0, rng=p[1].high-p[1].low;
      F.pivW_P=P; F.pivW_R1=2*P-p[1].low; F.pivW_S1=2*P-p[1].high;
      F.pivW_R2=P+rng; F.pivW_S2=P-rng;
      F.pivW_R3=p[1].high+2*(P-p[1].low); F.pivW_S3=p[1].low-2*(p[1].high-P);
   }
   double lv[40]; int n=0;
   for(int k=0;k<6;k++) lv[n++]=F.fibLevel[k];
   for(int k=0;k<4;k++) lv[n++]=F.fibExt[k];
   double dp[7]={F.pivD_P,F.pivD_R1,F.pivD_R2,F.pivD_S1,F.pivD_S2,F.pivD_R3,F.pivD_S3};
   for(int k=0;k<7;k++) lv[n++]=dp[k];
   double wp[3]={F.pivW_P,F.pivW_R1,F.pivW_S1};
   for(int k=0;k<3;k++) lv[n++]=wp[k];
   if(F.vwap>0) lv[n++]=F.vwap;
   if(F.obBullTop>0){lv[n++]=F.obBullTop;lv[n++]=F.obBullBot;}
   if(F.obBearTop>0){lv[n++]=F.obBearTop;lv[n++]=F.obBearBot;}
   if(F.fvgBullTop>0) lv[n++]=(F.fvgBullTop+F.fvgBullBot)/2;
   if(F.fvgBearTop>0) lv[n++]=(F.fvgBearTop+F.fvgBearBot)/2;
   int score=0; double ctol=0.25*atr;
   for(int i=0;i<n;i++) if(MathAbs(lv[i]-c1)<=ctol) score++;
   F.confluenceScore=score;
}

//====================== REGIME / MTF / SESSION / NEWS ==============
void UpdateRegimeAndMTF(){
   double c1=iClose(_Symbol,_Period,1);
   bool bullStack=(F.ema9>F.ema21&&F.ema21>F.ema50&&c1>F.ema50);
   bool bearStack=(F.ema9<F.ema21&&F.ema21<F.ema50&&c1<F.ema50);
   if(F.atrZ>2.0)                          F.regime=REGIME_HIGH_VOLATILITY;
   else if(F.adx>=25&&bullStack)           F.regime=REGIME_TRENDING_BULLISH;
   else if(F.adx>=25&&bearStack)           F.regime=REGIME_TRENDING_BEARISH;
   else if(F.bbwPct>0&&F.bbwPct<=30)       F.regime=REGIME_BREAKOUT;
   else if(F.adx<20&&(F.rsi>70||F.rsi<30)) F.regime=REGIME_MEAN_REVERSION;
   else                                    F.regime=REGIME_RANGE;

   ENUM_TIMEFRAMES mp[4]={PERIOD_M5,PERIOD_M15,PERIOD_H1,PERIOD_H4};
   F.mtfScore=0; F.mtfStates="";
   for(int i=0;i<4;i++){
      double e1=Buf(H.mtf[i],0,1),e4=Buf(H.mtf[i],0,4);
      double c=iClose(_Symbol,mp[i],1);
      int s=0; if(c>e1)s++; if(e1<e4)s--;
      F.mtfScore+=s;
      F.mtfStates+=StringFormat("%s%s ",StringSubstr(EnumToString(mp[i]),7),s>0?"+":s<0?"-":"0");
   }
   F.mtfScore/=6.0;

   MqlDateTime tm; TimeToStruct(TimeGMT(),tm);
   int h=tm.hour;
   F.isWeekend=(tm.day_of_week==0||tm.day_of_week==6)?1:0;
   F.isOverlap=0;
   if(h>=12&&h<16)     { F.session=SESS_OVERLAP; F.isOverlap=1; }
   else if(h>=7&&h<12)   F.session=SESS_LONDON;
   else if(h>=16&&h<21)  F.session=SESS_NEW_YORK;
   else if(h>=0&&h<7)    F.session=SESS_TOKYO;
   else if(h>=21)        F.session=SESS_SYDNEY;
   else                  F.session=SESS_OFF_HOURS;
}
void UpdateNewsRisk(int beforeMin=15,int afterMin=10){
   F.newsRisk=0;
   if(MQLInfoInteger(MQL_TESTER)) return;
   datetime now=TimeTradeServer();
   string ccys[2];
   ccys[0]=SymbolInfoString(_Symbol,SYMBOL_CURRENCY_BASE);
   ccys[1]=SymbolInfoString(_Symbol,SYMBOL_CURRENCY_PROFIT);
   for(int c=0;c<2;c++){
      MqlCalendarValue vals[];
      if(CalendarValueHistory(vals,now-(datetime)(afterMin*60),now+(datetime)(beforeMin*60),NULL,ccys[c])){
         int n=ArraySize(vals);
         for(int i=0;i<n;i++){
            MqlCalendarEvent ev;
            if(CalendarEventById(vals[i].event_id,ev)){
               if(ev.importance==CALENDAR_IMPORTANCE_HIGH) F.newsRisk+=1.0;
               else if(ev.importance==CALENDAR_IMPORTANCE_MODERATE) F.newsRisk+=0.3;
            }
         }
      }
   }
}

//====================== CANDLE INTELLIGENCE ========================
void UpdateCandleIntel(){
   MqlRates r[]; ArraySetAsSeries(r,true);
   if(CopyRates(_Symbol,_Period,0,60,r)<60) return;
   double o=r[1].open,h=r[1].high,l=r[1].low,c=r[1].close;
   double range=h-l; if(range<=0) range=_Point;
   double body=MathAbs(c-o);
   F.bodySize=body/_Point;
   F.upWick=(h-MathMax(o,c))/_Point; F.loWick=(MathMin(o,c)-l)/_Point;
   F.bodyRatio=body/range;
   double atr=(F.atr>0&&F.atr!=EMPTY_VALUE)?F.atr:range;
   F.atrNormRange=range/atr;
   F.patDoji    =(F.bodyRatio<=0.10)?1:0;
   F.patPinBull =(F.loWick>=2*(body/_Point)&&F.loWick>=0.6*(range/_Point))?1:0;
   F.patPinBear =(F.upWick>=2*(body/_Point)&&F.upWick>=0.6*(range/_Point))?1:0;
   F.rejection  =(F.patPinBull==1||F.patPinBear==1)?1:0;
   bool prevBear=(r[2].close<r[2].open), prevBull=(r[2].close>r[2].open);
   double pbody=MathAbs(r[2].close-r[2].open);
   F.patEngulfBull=(prevBear&&c>r[2].open&&o<=r[2].close&&body>pbody)?1:0;
   F.patEngulfBear=(prevBull&&c<r[2].open&&o>=r[2].close&&body>pbody)?1:0;
   F.patInside =(r[1].high<r[2].high&&r[1].low>r[2].low)?1:0;
   F.patOutside=(r[1].high>r[2].high&&r[1].low<r[2].low)?1:0;
   F.displBull=(c>o&&body>1.5*atr)?1:0;
   F.displBear=(c<o&&body>1.5*atr)?1:0;
   F.compression=(F.atrNormRange<0.6&&F.bodyRatio<0.4)?1:0;
   F.expansion  =(F.atrNormRange>1.5)?1:0;
   F.breakoutCandle=(((F.bbUp!=EMPTY_VALUE&&F.bbUp>0&&c>F.bbUp))||
                     ((F.bbLow!=EMPTY_VALUE&&F.bbLow>0&&c<F.bbLow)))?1:0;
   F.consecBull=F.consecBear=0;
   for(int i=1;i<59;i++){
      if(r[i].close>r[i].open){ if(F.consecBear>0)break; F.consecBull++; }
      else if(r[i].close<r[i].open){ if(F.consecBull>0)break; F.consecBear++; }
      else break;
   }
}

//====================== SCORING (weighted) =========================
void ScoreDirections(){
   double c1=iClose(_Symbol,_Period,1);
   double mid=(F.r150Hi+F.r150Lo)*0.5;
   double w_trend=1.2, w_struct=1.8, w_mom=0.8, w_vol=1.0, w_mtf=1.1, w_zone=1.5;
   for(int dir=-1;dir<=1;dir+=2){
      bool bull=(dir>0); double s=0;
      bool stack=bull?(F.ema9>F.ema21&&F.ema21>F.ema50):(F.ema9<F.ema21&&F.ema21<F.ema50);
      if(stack) s+=10*w_trend;
      if(bull?c1>F.ema200:c1<F.ema200) s+=6*w_trend;
      if(bull?F.macdHist>0:F.macdHist<0) s+=5*w_mom;
      if(bull?F.macdBullX==1:F.macdBearX==1) s+=3*w_mom;
      if(F.adx>=22&&(bull?F.plusDI>F.minusDI:F.minusDI>F.plusDI)) s+=8*w_trend;
      if(bull?F.ichimokuPos>0:F.ichimokuPos<0) s+=8*w_trend;
      if(F.sarLong==(bull?1:0)) s+=4*w_trend;
      if(bull?(F.emaCross921>0):(F.emaCross921<0)) s+=2;
      double ms=F.mtfScore;
      if(bull?ms>0.33:ms<-0.33) s+=10*w_mtf; else if(bull?ms>0:ms<0) s+=5*w_mtf;
      if(bull?F.structTrend==ST_BULL:F.structTrend==ST_BEAR) s+=8*w_struct;
      if(bull?F.bosBull==1:F.bosBear==1) s+=6*w_struct;
      if(bull?F.chochBull==1:F.chochBear==1) s+=6*w_struct;
      if(bull?F.sweepLow==1:F.sweepHigh==1) s+=8*w_struct;
      if(bull?c1<mid:c1>mid) s+=4*w_zone;
      bool zone=bull?
         ((F.obBullTop>0&&c1<=F.obBullTop&&c1>=F.obBullBot)||
          (F.fvgBullTop>0&&c1<=F.fvgBullTop&&c1>=F.fvgBullBot)||F.inGoldenZone==1):
         ((F.obBearTop>0&&c1<=F.obBearTop&&c1>=F.obBearBot)||
          (F.fvgBearTop>0&&c1<=F.fvgBearTop&&c1>=F.fvgBearBot)||F.inGoldenZone==1);
      if(zone) s+=6*w_zone;
      if(bull?(F.rsi>=45&&F.rsi<=68):(F.rsi>=32&&F.rsi<=55)) s+=4*w_mom;
      if(bull?F.stochK>F.stochD:F.stochK<F.stochD) s+=4*w_mom;
      if(bull?(F.srsiK>F.srsiD&&F.srsiK<75):(F.srsiK<F.srsiD&&F.srsiK>25)) s+=2;
      if(F.vwap>0&&(bull?c1>F.vwap:c1<F.vwap)) s+=6*w_vol;
      if(bull?F.bbBullRev==1:F.bbBearRev==1) s+=2;
      bool bkt=(bull&&F.breakoutCandle==1&&F.bbUp!=EMPTY_VALUE&&F.bbUp>0&&c1>F.bbUp)
             ||(!bull&&F.breakoutCandle==1&&F.bbLow!=EMPTY_VALUE&&F.bbLow>0&&c1<F.bbLow);
      if(bkt) s+=2;
      if(F.obvZ*(bull?1.0:-1.0)>0.5) s+=2;
      if(F.sessionHigh>0 && F.sessionLow>0){
         if(bull && c1 > F.sessionHigh - 0.2*(F.sessionHigh-F.sessionLow)) s += 3;
         if(!bull && c1 < F.sessionLow + 0.2*(F.sessionHigh-F.sessionLow)) s += 3;
      }
      if(F.tickDeltaZ*(bull?1.0:-1.0) > 0.3) s += 2;
      if(F.isOverlap==1) s+=4;
      else if(F.session==SESS_LONDON||F.session==SESS_NEW_YORK) s+=2;
      if(bull?F.regime==REGIME_TRENDING_BULLISH:F.regime==REGIME_TRENDING_BEARISH) s+=6*w_trend;
      else if(F.regime==REGIME_BREAKOUT&&MathAbs(ms)>0.33&&(bull?ms>0:ms<0)) s+=4*w_trend;
      if(bull?(F.patEngulfBull==1||F.patPinBull==1||F.displBull==1)
             :(F.patEngulfBear==1||F.patPinBear==1||F.displBear==1)) s+=4*w_mom;
      if(F.confluenceScore>=4) s+=6*w_zone; else if(F.confluenceScore>=2) s+=4*w_zone;
      if(dir>0) g_rawB=s; else g_rawS=s;
   }
   g_scoreB=(int)MathRound(g_rawB/SCORE_MAX*100.0);
   g_scoreS=(int)MathRound(g_rawS/SCORE_MAX*100.0);
}

//====================== DAILY STATS ================================
void UpdateDayStart(){
   datetime today = iTime(_Symbol, PERIOD_D1, 0);
   if(today != g_statsDay){
      g_statsDay = today;
      g_dayStartBalance = AccountInfoDouble(ACCOUNT_BALANCE);
      SaveState();
   }
}

void UpdateDayStats(){
   static uint lastMs=0;
   if(GetTickCount()-lastMs<2000) return;
   lastMs=GetTickCount();
   datetime d0=iTime(_Symbol,PERIOD_D1,0);
   HistorySelect(d0,TimeCurrent()+60);
   int tot=HistoryDealsTotal(),ins=0,n=0;
   ulong ids[256]; double sums[256];
   double pl=0;
   for(int i=0;i<tot;i++){
      ulong dt=HistoryDealGetTicket(i); if(dt==0) continue;
      if(HistoryDealGetInteger(dt,DEAL_MAGIC)!=InpMagic) continue;
      if(HistoryDealGetString(dt,DEAL_SYMBOL)!=_Symbol) continue;
      long e=HistoryDealGetInteger(dt,DEAL_ENTRY);
      double net=HistoryDealGetDouble(dt,DEAL_PROFIT)
                +HistoryDealGetDouble(dt,DEAL_SWAP)
                +HistoryDealGetDouble(dt,DEAL_COMMISSION);
      if(e==DEAL_ENTRY_IN){ ins++; continue; }
      ulong pid=(ulong)HistoryDealGetInteger(dt,DEAL_POSITION_ID);
      int f=-1;
      for(int k=0;k<n;k++) if(ids[k]==pid){f=k;break;}
      if(f<0&&n<256){ ids[n]=pid; sums[n]=0; f=n++; }
      if(f>=0) sums[f]+=net;
      pl+=net;
   }
   int w=0,l=0,sc=0;
   for(int k=0;k<n;k++){
      bool open=false;
      for(int p=PositionsTotal()-1;p>=0;p--){
         ulong t2=PositionGetTicket(p);
         if(t2>0&&(ulong)PositionGetInteger(POSITION_IDENTIFIER)==ids[k]){open=true;break;}
      }
      if(open) continue;
      if(sums[k]>0.0000001) w++; else if(sums[k]<-0.0000001) l++; else sc++;
   }
   g_dayTrades=ins; g_dayPL=pl; g_wins=w; g_losses=l; g_scratch=sc;
}

//====================== TRADE OPEN =================================
double GetMinStopForCost(double costPips, double pipSize){
   return (costPips / 0.15) * pipSize;
}

bool TryOpen(int dir){
   double pip=Pip();
   double c1=iClose(_Symbol,_Period,1);
   double atr=F.atr;
   double slDist=atr*InpSL_ATR;
   if(dir>0&&F.lastSwingLow>0){ double d=(c1-F.lastSwingLow)+0.3*atr; if(d>slDist) slDist=d; }
   if(dir<0&&F.lastSwingHigh>0){ double d=(F.lastSwingHigh-c1)+0.3*atr; if(d>slDist) slDist=d; }
   
   double costPips = (InpCostPips > 0) ? InpCostPips : A.spreadPips + (A.commissionPerLot / A.pipValuePerLot);
   double minStop = GetMinStopForCost(costPips, pip);
   if(slDist < minStop) slDist = minStop;
   
   if(slDist > g_maxSLPrice) {
      Reject("SKIP: SL exceeds profile maximum");
      return false;
   }
   if(slDist < g_minSLPrice) slDist = g_minSLPrice;

   double minDist = MinStopDist();
   if(slDist < minDist) slDist = minDist;

   double dynTP1=InpTP1_R;
   double tp1D=dynTP1*slDist;
   double price=(dir>0)?SymbolInfoDouble(_Symbol,SYMBOL_ASK):SymbolInfoDouble(_Symbol,SYMBOL_BID);
   double tpPrice = (dir>0)? price + tp1D : price - tp1D;
   double slPrice = (dir>0)? price - slDist : price + slDist;
   double tpDistancePoints = MathAbs(tpPrice - price)/_Point;
   double slDistancePoints = MathAbs(slPrice - price)/_Point;
   double minDistPoints = (double)MathMax(A.stopsLevel, A.freezeLevel);
   if(tpDistancePoints < minDistPoints || slDistancePoints < minDistPoints){
      Reject("SKIP: TP/SL too close (stops/freeze)");
      return false;
   }

   double eq=AccountInfoDouble(ACCOUNT_EQUITY);
   double riskMoney=eq*InpRiskPct/100.0;
   double maxRiskMoney=eq*InpMaxRiskPct/100.0;
   double pv=A.pipValuePerLot;
   if(pv<=0){ Reject("SKIP: pip value n/a"); return false; }
   double lots=NormLots(riskMoney/(slDist/pip*pv));
   double stopRisk = lots*(slDist/pip)*pv;
   double tradingCost = (lots * A.spreadCostPerLot) + (lots * A.commissionPerLot);
   double actualRisk = stopRisk + tradingCost;
   if(actualRisk > maxRiskMoney){
      Reject("SKIP: risk exceeds max allowed");
      return false;
   }
   if(lots <= 0){ Reject("SKIP: no lots"); return false; }

   double marginReq=0;
   if(!OrderCalcMargin(dir>0?ORDER_TYPE_BUY:ORDER_TYPE_SELL,_Symbol,lots,price,marginReq)
      ||marginReq>A.marginFree*0.8){
      Reject("SKIP: insufficient margin");
      return false;
   }
   trade.SetExpertMagicNumber(InpMagic);
   bool ok=(dir>0)
      ? trade.Buy (lots,_Symbol,0.0,NormalizeDouble(slPrice,_Digits),NormalizeDouble(tpPrice,_Digits),"PAT")
      : trade.Sell(lots,_Symbol,0.0,NormalizeDouble(slPrice,_Digits),NormalizeDouble(tpPrice,_Digits),"PAT");
   uint rc=trade.ResultRetcode();
   if(!ok||(rc!=TRADE_RETCODE_DONE&&rc!=TRADE_RETCODE_PLACED)){
      Reject(StringFormat("ORDER FAIL rc=%u",(uint)rc));
      return false;
   }
   double fill=trade.ResultPrice(); if(fill<=0) fill=price;
   T.active=true; T.ticket=FindMyTicket(); T.dir=dir;
   T.entry=fill; T.rDist=slDist; T.lots0=lots; T.openTime=TimeCurrent();
   T.sl=(dir>0)?T.entry-slDist:T.entry+slDist;
   T.tp1=T.entry+dir*InpTP1_R*slDist;
   T.tp2=0; T.tp3=0;
   T.beDone=false; T.tp1Done=false; T.tp2Done=false;
   T.riskMoney=actualRisk;
   SaveState();
   g_decision=StringFormat("OPENED %s %.2f lots (risk %.2f%%)",dir>0?"LONG":"SHORT",lots,actualRisk/eq*100);
   return true;
}

void EvaluateSetup(){
   g_decision="FLAT - scanning";
   if(!TerminalInfoInteger(TERMINAL_TRADE_ALLOWED)||!MQLInfoInteger(MQL_TRADE_ALLOWED)){
      Reject("BLOCKED: autotrading off"); return; }
   if(F.atr<=0||F.atr==EMPTY_VALUE){ Reject("WAIT: data warming"); return; }
   if(T.active){ g_decision="MANAGING position"; return; }
   ScoreDirections();
   long spr=SymbolInfoInteger(_Symbol,SYMBOL_SPREAD);
   if(F.isWeekend==1){ Reject("BLOCKED: weekend"); return; }
   if(InpTradeOnlyOverlap&&F.isOverlap==0){ Reject("BLOCKED: session"); return; }
   if(InpUseNewsFilter&&F.newsRisk>0){
      Reject(StringFormat("BLOCKED: news %.1f",F.newsRisk)); return; }
   if(spr>g_spreadLimit){
      Reject(StringFormat("BLOCKED: spread %d>%d",(int)spr,g_spreadLimit)); return; }
   UpdateDayStart();
   UpdateDayStats();
   if(g_dayTrades>=InpMaxTradesPerDay){ Reject("BLOCKED: max trades/day"); return; }
   double dailyLossLimit = g_dayStartBalance * InpDailyLossCapPct / 100.0;
   if(g_dayPL <= -dailyLossLimit){
      Reject("BLOCKED: daily loss cap"); return; }
   int best=MathMax(g_scoreB,g_scoreS);
   int dir =(g_scoreB>=g_scoreS)?1:-1;
   int oth =(dir>0)?g_scoreS:g_scoreB;
   g_bestDir=dir; g_bestScore=best;
   if(best<g_scoreThreshold){
      Reject(StringFormat("WAIT: %s%d vs %d  <%d",dir>0?"B":"S",best,oth,(int)g_scoreThreshold));
      return;
   }
   if(best-oth<InpScoreMargin){
      Reject(StringFormat("WAIT: margin %d<%d",best-oth,InpScoreMargin));
      return;
   }
   TryOpen(dir);
}

//====================== TRADE MANAGEMENT ===========================
bool CloseVol(ulong tk,double want){
   if(!PositionSelectByTicket(tk)) return false;
   double vol=PositionGetDouble(POSITION_VOLUME);
   double mn=SymbolInfoDouble(_Symbol,SYMBOL_VOLUME_MIN);
   want=NormLots(want);
   if(want<mn) return true;
   if(vol-want<mn){
      bool ok=trade.PositionClose(tk);
      if(ok) T.active=false;
      return ok;
   }
   return trade.PositionClosePartial(tk,want);
}
void Adopt(ulong tk){
   if(!PositionSelectByTicket(tk)) return;
   T.active=true; T.ticket=tk;
   T.dir=(PositionGetInteger(POSITION_TYPE)==POSITION_TYPE_BUY)?1:-1;
   T.entry=PositionGetDouble(POSITION_PRICE_OPEN);
   T.openTime=(datetime)PositionGetInteger(POSITION_TIME);
   T.lots0=PositionGetDouble(POSITION_VOLUME);
   double sl=PositionGetDouble(POSITION_SL);
   T.rDist=(sl>0)?MathAbs(T.entry-sl):((F.atr>0)?F.atr*InpSL_ATR:100*_Point);
   T.tp1=PositionGetDouble(POSITION_TP);
   T.tp2=0; T.tp3=0;
   LoadState();
   if(T.entry<=0 || MathAbs(T.entry - PositionGetDouble(POSITION_PRICE_OPEN)) > 10*_Point){
      T.beDone=false;
      T.tp1Done=false;
      T.tp2Done=false;
   }
   SaveState();
}
void ManagePosition(){
   ulong tk=FindMyTicket();
   if(tk==0){
      if(T.active){ T.active=false; SaveState(); }
      g_floating=0; g_Rmultiple=0; return;
   }
   if(!T.active) Adopt(tk);
   if(!PositionSelectByTicket(tk)) return;
   double bid=SymbolInfoDouble(_Symbol,SYMBOL_BID);
   double ask=SymbolInfoDouble(_Symbol,SYMBOL_ASK);
   double price=(T.dir>0)?bid:ask;
   double curSL=PositionGetDouble(POSITION_SL);
   double curTP=PositionGetDouble(POSITION_TP);
   double pip=Pip();
   double minDist=MinStopDist();
   double profitDist=(price-T.entry)*T.dir;
   g_floating=PositionGetDouble(POSITION_PROFIT);
   g_Rmultiple=(T.rDist>0)?profitDist/T.rDist:0;
   bool changed=false;

   if(!T.beDone && profitDist >= InpTrailStart_R * T.rDist){
      double beCostPips = MathMax(InpCostPips, A.spreadPips);
      double be = T.entry + T.dir * (beCostPips + InpBE_BufferPips) * pip;
      bool valid=(T.dir>0)?(bid-be>=minDist):(be-ask>=minDist);
      if(valid){
         if(trade.PositionModify(tk,NormalizeDouble(be,_Digits),curTP)){
            T.beDone=true; changed=true;
         }
      }
   }
   if(InpUseTrail && profitDist >= InpTrailStart_R * T.rDist){
      double atr=(F.atr>0&&F.atr!=EMPTY_VALUE)?F.atr:T.rDist;
      double ns=(T.dir>0)?price-InpTrailStep_ATR*atr:price+InpTrailStep_ATR*atr;
      bool better=(T.dir>0)?(ns>curSL+_Point):(curSL==0||ns<curSL-_Point);
      bool valid =(T.dir>0)?(bid-ns>=minDist):(ns-ask>=minDist);
      if(better&&valid&&trade.PositionModify(tk,NormalizeDouble(ns,_Digits),curTP)) changed=true;
   }
   if(InpMaxBarsInTrade>0){
      int barsIn=iBarShift(_Symbol,_Period,T.openTime);
      if(barsIn>=InpMaxBarsInTrade && profitDist<0.2*T.rDist){
         if(trade.PositionClose(tk)){ T.active=false; changed=true; }
      }
   }
   if(InpMaxSecondsInTrade>0 && T.active){
      if((TimeCurrent() - T.openTime) >= InpMaxSecondsInTrade && profitDist<0.2*T.rDist){
         if(trade.PositionClose(tk)){ T.active=false; changed=true; }
      }
   }
   if(changed) SaveState();
}

//====================== DASHBOARD ==================================
int PX,PY; const int PW=400,ROWH=18;
#define ROWS_MAIN   23
#define ROWS_ACC    12
color CLR_BG=C'14,18,26',CLR_HDR=C'26,33,46',CLR_BRD=C'45,55,72',CLR_SEP=C'45,55,72';
color CLR_TXT=C'212,218,230',CLR_DIM=C'120,130,148';
color CLR_GRN=C'0,220,130',CLR_RED=C'255,90,90',CLR_AMB=C'255,190,70',CLR_CYN=C'0,190,215';

void Rect(string n,int x,int y,int w,int h,color bg,color br){
   ObjectCreate(0,n,OBJ_RECTANGLE_LABEL,0,0,0);
   ObjectSetInteger(0,n,OBJPROP_CORNER,CORNER_LEFT_UPPER);
   ObjectSetInteger(0,n,OBJPROP_XDISTANCE,x); ObjectSetInteger(0,n,OBJPROP_YDISTANCE,y);
   ObjectSetInteger(0,n,OBJPROP_XSIZE,w);     ObjectSetInteger(0,n,OBJPROP_YSIZE,h);
   ObjectSetInteger(0,n,OBJPROP_BGCOLOR,bg);
   ObjectSetInteger(0,n,OBJPROP_BORDER_TYPE,BORDER_FLAT);
   ObjectSetInteger(0,n,OBJPROP_COLOR,br);
   ObjectSetInteger(0,n,OBJPROP_BACK,false);
   ObjectSetInteger(0,n,OBJPROP_SELECTABLE,false);
   ObjectSetInteger(0,n,OBJPROP_HIDDEN,true);
   ObjectSetInteger(0,n,OBJPROP_ZORDER,0);
}
void Lbl(string n,int x,int y,string t,int sz,color c,string f="Consolas",
         ENUM_ANCHOR_POINT a=ANCHOR_LEFT_UPPER){
   if(ObjectFind(0,n)<0) ObjectCreate(0,n,OBJ_LABEL,0,0,0);
   ObjectSetInteger(0,n,OBJPROP_CORNER,CORNER_LEFT_UPPER);
   ObjectSetInteger(0,n,OBJPROP_XDISTANCE,x); ObjectSetInteger(0,n,OBJPROP_YDISTANCE,y);
   ObjectSetString (0,n,OBJPROP_TEXT,t);
   ObjectSetInteger(0,n,OBJPROP_FONTSIZE,sz);
   ObjectSetInteger(0,n,OBJPROP_COLOR,c);
   ObjectSetString (0,n,OBJPROP_FONT,f);
   ObjectSetInteger(0,n,OBJPROP_ANCHOR,a);
   ObjectSetInteger(0,n,OBJPROP_SELECTABLE,false);
   ObjectSetInteger(0,n,OBJPROP_HIDDEN,true);
}
void SetRow(int i,string l,string v,color vc){
   Lbl(PFX+"L"+(string)i,PX+12,PY+36+i*ROWH,l,9,CLR_DIM);
   Lbl(PFX+"V"+(string)i,PX+PW-12,PY+36+i*ROWH,v,9,vc,"Consolas",ANCHOR_RIGHT_UPPER);
}
void BuildDashboard(){
   PX=InpPanelX; PY=InpPanelY;
   int rows=InpShowAccount?ROWS_MAIN+ROWS_ACC:ROWS_MAIN;
   int PH=36+rows*ROWH+8;
   Rect(PFX+"bg",PX,PY,PW,PH,CLR_BG,CLR_BRD);
   Rect(PFX+"hdr",PX,PY,PW,28,CLR_HDR,CLR_HDR);
   Lbl(PFX+"title",PX+12,PY+6,"PREDICT-A-TRADE v1.2",11,CLR_CYN,"Arial Black");
   Rect(PFX+"dot",PX+PW-20,PY+10,8,8,CLR_AMB,CLR_AMB);
   int barRow=13;
   Rect(PFX+"barbg",PX+95,PY+36+barRow*ROWH+3,PW-170,11,CLR_HDR,CLR_HDR);
   Rect(PFX+"bar",PX+96,PY+36+barRow*ROWH+4,1,9,CLR_CYN,CLR_CYN);
   if(InpShowAccount){
      int sepY=PY+36+ROWS_MAIN*ROWH+2;
      Rect(PFX+"sep",PX+8,sepY,PW-16,1,CLR_SEP,CLR_SEP);
   }
   Lbl(PFX+"foot",PX+12,PY+36+rows*ROWH-4,
       "Predict-A-Trade | https://predictatrade.com",7,CLR_DIM);
   ChartRedraw();
}
string TfStr(){ string s=EnumToString(_Period); return StringSubstr(s,7); }
string SessStr(){
   string s[]={"OFF","SYDNEY","TOKYO","LONDON","NEWYORK","OVERLAP"};
   return s[(int)F.session];
}
string RegStr(){
   string s[]={"NONE","TREND-BULL","TREND-BEAR","RANGE","MEAN-REV","BREAKOUT","HI-VOL"};
   return s[(int)F.regime];
}
void UpdateDashboard(){
   color dotC=CLR_AMB;
   if(StringFind(g_decision,"BLOCKED")==0||StringFind(g_decision,"SKIP")==0) dotC=CLR_RED;
   else if(StringFind(g_decision,"OPENED")==0||StringFind(g_decision,"MANAGING")==0) dotC=CLR_GRN;
   ObjectSetInteger(0,PFX+"dot",OBJPROP_BGCOLOR,dotC);
   ObjectSetInteger(0,PFX+"dot",OBJPROP_COLOR,dotC);

   MqlDateTime g; TimeToStruct(TimeGMT(),g);
   double pip=Pip();
   double c1=iClose(_Symbol,_Period,1);

   g_drv = StringFormat("B%d/S%d ADX%.1f MTF%.2f CF%.0f",
                        g_scoreB, g_scoreS, F.adx, F.mtfScore, F.confluenceScore);

   SetRow(0,"Symbol / Profile",_Symbol+" "+TfStr()+" / "+g_profileName,CLR_TXT);
   SetRow(1,"Session / GMT",
      StringFormat("%s%s %02d:%02d",SessStr(),F.isOverlap==1?" *":"",g.hour,g.min),
      F.isOverlap==1?CLR_GRN:CLR_DIM);
   SetRow(2,"Regime",RegStr(),
      F.regime==REGIME_TRENDING_BULLISH?CLR_GRN:
      F.regime==REGIME_TRENDING_BEARISH?CLR_RED:CLR_AMB);
   SetRow(3,"Spread / Limit",
      StringFormat("%d / %d pts",(int)A.spreadPts,g_spreadLimit),
      A.spreadPts>g_spreadLimit?CLR_RED:CLR_GRN);
   SetRow(4,"Session H/L",
      (F.sessionHigh>0)?StringFormat("%.5f / %.5f",F.sessionHigh,F.sessionLow):"-",
      CLR_TXT);
   SetRow(5,"Tick Delta Z",
      StringFormat("%+.2f",F.tickDeltaZ),
      F.tickDeltaZ>0?CLR_GRN:F.tickDeltaZ<0?CLR_RED:CLR_DIM);
   SetRow(6,"Trend / ADX",
      StringFormat("%s | ADX %.0f +%.0f -%.0f",(F.structTrend==ST_BULL?"BULL":F.structTrend==ST_BEAR?"BEAR":"RANGE"),F.adx,F.plusDI,F.minusDI),
      (F.adx>25)?CLR_GRN:CLR_AMB);
   SetRow(7,"Momentum",
      StringFormat("RSI %.1f St %.0f/%.0f MACD%s",F.rsi,F.stochK,F.stochD,F.macdHist>=0?"+":"-"),
      (F.rsi>70||F.rsi<30)?CLR_AMB:CLR_TXT);
   SetRow(8,"Structure",
      StringFormat("BOS%s%s CH%s%s SW%s%s",F.bosBull==1?"+B":"",F.bosBear==1?"+S":"",
         F.chochBull==1?"+B":"",F.chochBear==1?"+S":"",
         F.sweepLow==1?"-L":"",F.sweepHigh==1?"-H":""),
      (F.bosBull+F.bosBear+F.chochBull+F.chochBear)>0?CLR_CYN:CLR_DIM);
   SetRow(9,"Zones / CF",
      StringFormat("GZ%d CF%.0f",F.inGoldenZone,F.confluenceScore),
      F.confluenceScore>=2?CLR_CYN:CLR_DIM);
   SetRow(10,"VWAP",
      (F.vwap>0)?StringFormat("%+.1fp %s",(c1-F.vwap)/pip,c1>F.vwap?"above":"below"):"-",
      (F.vwap>0&&c1>F.vwap)?CLR_GRN:CLR_RED);
   SetRow(11,"Candle",
      StringFormat("B%d/S%d %s%s%s",F.consecBull,F.consecBear,
         F.patEngulfBull==1?"EGB ":F.patEngulfBear==1?"EGS ":"",
         F.patPinBull==1?"PB ":F.patPinBear==1?"PS ":"",
         F.displBull==1||F.displBear==1?"DIS":""),
      CLR_TXT);
   SetRow(12,"Drivers",g_drv,CLR_TXT);

   int best=MathMax(g_scoreB,g_scoreS);
   double frac=MathMax(0.0,MathMin(1.0,best/100.0));
   ObjectSetInteger(0,PFX+"bar",OBJPROP_XSIZE,(int)((PW-172)*frac));
   ObjectSetInteger(0,PFX+"bar",OBJPROP_BGCOLOR,best>=g_scoreThreshold?CLR_GRN:CLR_CYN);
   SetRow(13,"Score",
      StringFormat("B%d S%d (need %d)",g_scoreB,g_scoreS,(int)g_scoreThreshold),
      best>=g_scoreThreshold?CLR_GRN:CLR_AMB);
   SetRow(14,"Signal",
      StringFormat("%s %d vs %d",g_bestDir==0?"-":g_bestDir>0?"LONG":"SHORT",g_bestScore,
                   g_bestDir>0?g_scoreS:g_scoreB),
      g_bestDir>0?CLR_GRN:CLR_RED);
   SetRow(15,"Decision",g_decision,
      StringFind(g_decision,"BLOCKED")==0?CLR_RED:
      StringFind(g_decision,"WAIT")==0?CLR_AMB:
      StringFind(g_decision,"SKIP")==0?CLR_RED:
      StringFind(g_decision,"OPENED")==0?CLR_GRN:
      StringFind(g_decision,"MANAGING")==0?CLR_GRN:CLR_DIM);
   SetRow(16,"Position",
      T.active?StringFormat("%s %.2f @ %s %+.2fR",
                T.dir>0?"LONG":"SHORT",T.lots0,DoubleToString(T.entry,_Digits),g_Rmultiple):"FLAT",
      T.active?(T.dir>0?CLR_GRN:CLR_RED):CLR_DIM);
   double liveSL=0;
   if(T.active&&PositionSelectByTicket(T.ticket)) liveSL=PositionGetDouble(POSITION_SL);
   SetRow(17,"Stop",
      T.active?StringFormat("%s %s",DoubleToString(liveSL,_Digits),
                T.beDone?"[BE-NO-LOSS]":"[initial]"):"-",
      T.beDone?CLR_GRN:CLR_AMB);
   SetRow(18,"Target",
      T.active?StringFormat("TP %.5f",T.tp1):"-",
      CLR_AMB);
   SetRow(19,"Trade P/L",
      T.active?StringFormat("%+.2f %s",g_floating,A.currency):"-",
      g_floating>0?CLR_GRN:g_floating<0?CLR_RED:CLR_DIM);
   SetRow(20,"Today",
      StringFormat("%d trades  %+.2f %s",g_dayTrades,g_dayPL,A.currency),
      g_dayPL>=0?CLR_GRN:CLR_RED);
   int closed=g_wins+g_losses;
   double wr=(closed>0)?100.0*g_wins/closed:0;
   SetRow(21,"Quality",
      StringFormat("W%d L%d S%d  WR %.0f%%",g_wins,g_losses,g_scratch,wr),
      wr>=60?CLR_GRN:wr>0?CLR_AMB:CLR_DIM);
   double capUsed = g_dayStartBalance * InpDailyLossCapPct / 100.0;
   SetRow(22,"Risk Guard",
      StringFormat("risk %.2f%% cap %.0f%% spr %dpt",
         InpRiskPct,
         (capUsed>0)?MathMax(0.0,MathMin(100.0,-g_dayPL/capUsed*100.0)):0,
         (int)A.spreadPts),
      (-g_dayPL)>capUsed*0.6?CLR_RED:CLR_TXT);

   if(InpShowAccount){
      SetRow(23,"== ACCOUNT / BROKER ==","",CLR_CYN);
      SetRow(24,"Account Type",A.accountType,CLR_TXT);
      SetRow(25,"Leverage",
         StringFormat("1:%d | margin lvl %s",(int)A.leverage,
            A.marginLevel>0?StringFormat("%.0f%%",A.marginLevel):"-"),
         A.marginLevel>0&&A.marginLevel<200?CLR_RED:CLR_TXT);
      SetRow(26,"Spread",
         StringFormat("%d pt (%.1f pips) = %s %.2f/lot",
            (int)A.spreadPts,A.spreadPips,A.currency,A.spreadCostPerLot),
         A.spreadPts>g_spreadLimit?CLR_RED:CLR_GRN);
      SetRow(27,"Commission",
         StringFormat("%.2f %s / lot",A.commissionPerLot,A.currency),CLR_TXT);
      SetRow(28,"Point / Pip value",
         StringFormat("%s %.2f /pt | %s %.2f /pip",
            A.currency,A.pointValuePerLot,A.currency,A.pipValuePerLot),
         CLR_TXT);
      SetRow(29,"Digits / Point / CSize",
         StringFormat("%d | %s | %s",
            A.digits,DoubleToString(A.point,A.digits),
            DoubleToString(A.contractSize,0)),
         CLR_TXT);
      SetRow(30,"Lots min/step/max",
         StringFormat("%.2f / %.2f / %.2f",A.volMin,A.volStep,A.volMax),CLR_TXT);
      SetRow(31,"Stops / Freeze",
         StringFormat("%d / %d pt",(int)A.stopsLevel,(int)A.freezeLevel),
         (A.stopsLevel+A.freezeLevel)>0?CLR_AMB:CLR_DIM);
      SetRow(32,"Swap L / S",
         StringFormat("%.1f / %.1f pts",A.swapLong,A.swapShort),
         (A.swapLong<0&&A.swapShort<0)?CLR_AMB:CLR_TXT);
      SetRow(33,"Balance / Equity",
         StringFormat("%.2f / %.2f %s",A.balance,A.equity,A.currency),
         A.equity>=A.balance?CLR_GRN:CLR_RED);
      SetRow(34,"Margin free/used",
         StringFormat("%.2f | %.2f (%.1f%% eq)",
            A.marginFree,A.marginUsed,
            (A.equity>0)?100.0*A.marginUsed/A.equity:0),
         (A.equity>0&&A.marginUsed/A.equity>0.3)?CLR_RED:CLR_TXT);
   }
   ChartRedraw();
}

//====================== LIFECYCLE ==================================
void SetOptimalFillingAndSlippage(){
   int dynamicSlippage = (InpSlippagePoints > 0) ? InpSlippagePoints : 10 + (int)(A.spreadPts * 0.5);
   trade.SetDeviationInPoints(dynamicSlippage);
   int filling = (int)SymbolInfoInteger(_Symbol, SYMBOL_FILLING_MODE);
   if((filling & SYMBOL_FILLING_FOK) == SYMBOL_FILLING_FOK) {
      trade.SetTypeFilling(ORDER_FILLING_FOK);
   } else if((filling & SYMBOL_FILLING_IOC) == SYMBOL_FILLING_IOC) {
      trade.SetTypeFilling(ORDER_FILLING_IOC);
   } else {
      trade.SetTypeFilling(ORDER_FILLING_RETURN);
   }
}

bool IsRolloverTime(){
   MqlDateTime dt; TimeToStruct(TimeTradeServer(), dt);
   if(dt.hour == 23 && dt.min >= 15) return true;
   if(dt.hour == 0 && dt.min <= 45) return true;
   return false;
}

void CheckEmergencyCapitalProtection(){
   double equity = AccountInfoDouble(ACCOUNT_EQUITY);
   double balance = AccountInfoDouble(ACCOUNT_BALANCE);
   double maxFloatingDrawdownPct = 5.0;
   double floatingLoss = balance - equity;
   if(floatingLoss > balance * maxFloatingDrawdownPct / 100.0){
      Print("EMERGENCY STOP: Floating drawdown exceeded ", maxFloatingDrawdownPct, "%. Closing all positions.");
      // Close all positions for this symbol
      for(int i=PositionsTotal()-1; i>=0; i--){
         ulong tk = PositionGetTicket(i);
         if(tk > 0 && PositionGetString(POSITION_SYMBOL) == _Symbol && PositionGetInteger(POSITION_MAGIC) == InpMagic){
            trade.PositionClose(tk);
         }
      }
      T.active = false;
      SaveState();
   }
}

int OnInit(){
   if(!FeaturesInit()){ Print("Predict-A-Trade: handle init failed"); return INIT_FAILED; }
   trade.SetExpertMagicNumber(InpMagic);
   trade.SetTypeFillingBySymbol(_Symbol);
   UpdateAccountInfo();
   SetSymbolProfile();
   SetOptimalFillingAndSlippage();
   LoadState();
   UpdateDayStart();
   Print("=== Predict-A-Trade v1.2 | ",_Symbol," | Profile: ",g_profileName," ===");
   Print("  Website: https://predictatrade.com");
   Print("  Account Type: ",A.accountType);
   Print("  Commission per lot: ",DoubleToString(A.commissionPerLot,2)," ",A.currency);
   Print(StringFormat("  Min SL %.5f, Max SL %.5f, Spread limit %d, Score threshold %d",
      g_minSLPrice,g_maxSLPrice,g_spreadLimit,(int)g_scoreThreshold));
   ZeroMemory(T);
   LoadState();
   BuildDashboard();
   return INIT_SUCCEEDED;
}
void OnDeinit(const int reason){
   ObjectsDeleteAll(0,PFX);
   ChartRedraw();
}
void OnTick(){
   if(SymbolInfoDouble(_Symbol,SYMBOL_BID)<=0) return;
   UpdateAccountInfo();
   CheckEmergencyCapitalProtection();
   ManagePosition();
   if(IsNewBar()){
      UpdateHandleIndicators();
      UpdateStochRSI();
      UpdateSessionHighLowAndVolumeDelta();
      UpdateVolatilityExtras();
      UpdateVolumeFeatures();
      UpdateStructure();
      UpdateFibAndPivots();
      UpdateRegimeAndMTF();
      UpdateNewsRisk();
      UpdateCandleIntel();
      UpdateDayStart();
      UpdateDayStats();
      EvaluateSetup();
   }
   if(GetTickCount()-g_lastPanel>300){
      g_lastPanel=GetTickCount();
      UpdateDashboard();
   }
}