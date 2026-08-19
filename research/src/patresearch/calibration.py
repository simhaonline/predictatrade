"""Calibration module — SOW Section 16, 134.6
Brier score, ECE, Wilson intervals, sigmoid calibration.
"""
import numpy as np
from scipy.special import expit as sigmoid

def brier_score(probabilities, outcomes):
    """Brier score — SOW 134.6"""
    probs = np.array(probabilities)
    outs = np.array([1.0 if o else 0.0 for o in outcomes])
    return float(np.mean((probs - outs) ** 2))

def ece(probabilities, outcomes, n_bins=10):
    """Expected Calibration Error — SOW 134.6"""
    probs = np.array(probabilities)
    outs = np.array([1.0 if o else 0.0 for o in outcomes])
    bin_edges = np.linspace(0, 1, n_bins + 1)
    
    total_ece = 0.0
    n = len(probs)
    for i in range(n_bins):
        mask = (probs >= bin_edges[i]) & (probs < bin_edges[i + 1])
        if mask.sum() == 0:
            continue
        bin_mean = probs[mask].mean()
        bin_observed = outs[mask].mean()
        total_ece += (mask.sum() / n) * abs(bin_observed - bin_mean)
    return total_ece

def wilson_interval(successes, total, z=1.96):
    """Wilson score interval — SOW 134.7"""
    if total == 0:
        return 0.0, 1.0
    n = float(total)
    p = successes / n
    z2 = z * z
    denom = 1 + z2 / n
    center = (p + z2 / (2 * n)) / denom
    spread = z * np.sqrt(p * (1 - p) / n + z2 / (4 * n * n)) / denom
    return float(center - spread), float(center + spread)

def sample_sufficiency(observed_p, margin=0.05, confidence=0.95):
    """Minimum sample size for desired margin — SOW 134.8"""
    z = {0.90: 1.645, 0.95: 1.96, 0.99: 2.576}.get(confidence, 1.96)
    p = observed_p if 0 < observed_p < 1 else 0.5
    n = (z ** 2 * p * (1 - p)) / (margin ** 2)
    return int(np.ceil(n))

def fit_sigmoid_calibration(raw_scores, outcomes):
    """Fit sigmoid calibration: calibrate(x) = sigmoid(a*x + b)"""
    from scipy.optimize import minimize
    scores = np.array(raw_scores) / 100.0  # normalize to 0-1
    outs = np.array([1.0 if o else 0.0 for o in outcomes])
    
    def loss(params):
        a, b = params
        calibrated = sigmoid(a * scores + b)
        return np.mean((calibrated - outs) ** 2)
    
    result = minimize(loss, [2.0, -0.5], method='Nelder-Mead')
    return {'a': float(result.x[0]), 'b': float(result.x[1])}

def apply_calibration(raw_score, params):
    """Apply sigmoid calibration to a raw score."""
    a, b = params['a'], params['b']
    return float(sigmoid(a * (raw_score / 100.0) + b))
