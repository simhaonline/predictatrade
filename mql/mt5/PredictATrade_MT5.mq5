//+------------------------------------------------------------------+
//|                                          PredictATrade_MT5.mq5   |
//|                            Predict-A-Trade v1.0.0                |
//|              Licensed signal reception and execution EA          |
//+------------------------------------------------------------------+
#property copyright "Predict-A-Trade"
#property version   "1.00"

// SOW Section 45: MT5 EA with CTrade class for modern order management
// Same signal protocol as MT4 (SOW Section 49)

#include <Trade\Trade.mqh>

input string  AgentPipeName = "\\\\.\\pipe\\PredictATradeMT5";
input bool    AutoExecute = false;
input ulong   MagicNumber = 20240001;

CTrade        trade;
string        g_currentDirection = "NONE";
string        g_connection = "OFFLINE";

//+------------------------------------------------------------------+
//| Expert initialization function                                   |
//+------------------------------------------------------------------+
int OnInit()
{
    Print("Predict-A-Trade MT5 EA v1.0.0 initialized");
    trade.SetExpertMagicNumber(MagicNumber);
    return(INIT_SUCCEEDED);
}

//+------------------------------------------------------------------+
//| Expert deinitialization function                                 |
//+------------------------------------------------------------------+
void OnDeinit(const int reason)
{
    Print("Predict-A-Trade MT5 EA deinitialized: ", reason);
}

//+------------------------------------------------------------------+
//| Expert tick function                                             |
//+------------------------------------------------------------------+
void OnTick()
{
    // TODO: Read from named pipe, parse signal, validate, execute
    UpdatePanel();
}

//+------------------------------------------------------------------+
//| Update EA Panel                                                  |
//+------------------------------------------------------------------+
void UpdatePanel()
{
    string panel = "Predict-A-Trade MT5\n";
    panel += "Connection: " + g_connection + "\n";
    panel += "Mode: " + (AutoExecute ? "AUTO" : "SIGNAL ONLY") + "\n";
    panel += "Signal: " + g_currentDirection + "\n";
    Comment(panel);
}
