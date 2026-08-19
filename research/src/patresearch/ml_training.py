"""ML training pipeline for adaptation models.

CRITICAL: Training runs OFFLINE in the research plane.
Only inference runs in the Go production hot path.

Data leakage protection:
- Chronological split (train → validation → test)
- Walk-forward validation
- Feature timestamps precede prediction outcomes
- Never randomly mix future observations into past training
"""
from dataclasses import dataclass, field
from datetime import datetime
from typing import List, Dict, Optional, Tuple
import numpy as np


@dataclass
class TrainingConfig:
    """Configuration for ML model training."""
    model_name: str = "adaptation_v1"
    min_samples: int = 100
    train_ratio: float = 0.6
    validation_ratio: float = 0.2
    test_ratio: float = 0.2
    walk_forward_folds: int = 5
    random_state: int = 42
    feature_columns: List[str] = field(default_factory=lambda: [
        "regime", "confluence", "confidence", "manipulation_index",
        "volatility", "liquidity_score", "spread", "atr",
        "session", "recent_returns", "sentiment_score"
    ])
    target_columns: List[str] = field(default_factory=lambda: [
        "stop_distance_multiplier", "position_size_multiplier", "minimum_confluence"
    ])


@dataclass
class TrainingResult:
    """Result of a model training run."""
    model_name: str
    version: str
    trained_at: str
    sample_count: int
    train_samples: int
    validation_samples: int
    test_samples: int
    metrics: Dict[str, float] = field(default_factory=dict)
    feature_schema: str = ""
    artifact_path: str = ""
    checksum: str = ""


def chronological_split(
    features: np.ndarray,
    targets: np.ndarray,
    config: TrainingConfig
) -> Tuple[Tuple[np.ndarray, np.ndarray], Tuple[np.ndarray, np.ndarray], Tuple[np.ndarray, np.ndarray]]:
    """Split data chronologically to prevent leakage.

    Order: train (earliest) → validation → test (latest)
    NEVER randomly shuffle — that would leak future into past.
    """
    n = len(features)
    if n < config.min_samples:
        raise ValueError(f"Insufficient samples: {n} < {config.min_samples}")

    train_end = int(n * config.train_ratio)
    val_end = int(n * (config.train_ratio + config.validation_ratio))

    train_x, train_y = features[:train_end], targets[:train_end]
    val_x, val_y = features[train_end:val_end], targets[train_end:val_end]
    test_x, test_y = features[val_end:], targets[val_end:]

    return (train_x, train_y), (val_x, val_y), (test_x, test_y)


def walk_forward_split(
    features: np.ndarray,
    targets: np.ndarray,
    n_folds: int
) -> List[Tuple[Tuple[np.ndarray, np.ndarray], Tuple[np.ndarray, np.ndarray]]]:
    """Walk-forward validation splits.

    Each fold trains on earlier data and validates on later data.
    This prevents future leakage.
    """
    n = len(features)
    fold_size = n // (n_folds + 1)

    folds = []
    for i in range(n_folds):
        train_end = fold_size * (i + 1)
        val_start = train_end
        val_end = min(val_start + fold_size, n)

        train_x, train_y = features[:train_end], targets[:train_end]
        val_x, val_y = features[val_start:val_end], targets[val_start:val_end]

        folds.append(((train_x, train_y), (val_x, val_y)))

    return folds


def validate_no_leakage(
    feature_timestamps: List[datetime],
    outcome_timestamps: List[datetime]
) -> bool:
    """Validate that feature timestamps precede outcome timestamps.

    Returns True if no leakage detected.
    """
    for ft, ot in zip(feature_timestamps, outcome_timestamps):
        if ft >= ot:
            return False  # feature is from after or same time as outcome = leakage
    return True


def train_model(
    features: np.ndarray,
    targets: np.ndarray,
    config: TrainingConfig,
    model_class: str = "random_forest"
) -> TrainingResult:
    """Train an ML model with leakage protection.

    Uses sklearn if available. Does not require XGBoost.
    Training is OFFLINE — never called from the production hot path.
    """
    from sklearn.ensemble import RandomForestRegressor
    from sklearn.metrics import mean_squared_error, r2_score

    # Chronological split
    (train_x, train_y), (val_x, val_y), (test_x, test_y) = chronological_split(
        features, targets, config
    )

    # Train model
    if model_class == "random_forest":
        model = RandomForestRegressor(
            n_estimators=100,
            random_state=config.random_state,
            n_jobs=-1
        )
    else:
        model = RandomForestRegressor(
            n_estimators=50,
            random_state=config.random_state
        )

    model.fit(train_x, train_y)

    # Validate
    val_pred = model.predict(val_x)
    val_mse = mean_squared_error(val_y, val_pred)
    val_r2 = r2_score(val_y, val_pred, multioutput="variance_weighted")

    # Test (out-of-sample)
    test_pred = model.predict(test_x)
    test_mse = mean_squared_error(test_y, test_pred)
    test_r2 = r2_score(test_y, test_pred, multioutput="variance_weighted")

    version = f"{datetime.utcnow().strftime('%Y%m%d%H%M%S')}"

    result = TrainingResult(
        model_name=config.model_name,
        version=version,
        trained_at=datetime.utcnow().isoformat(),
        sample_count=len(features),
        train_samples=len(train_x),
        validation_samples=len(val_x),
        test_samples=len(test_x),
        metrics={
            "val_mse": float(val_mse),
            "val_r2": float(val_r2),
            "test_mse": float(test_mse),
            "test_r2": float(test_r2),
        },
        feature_schema=",".join(config.feature_columns),
    )

    return result
