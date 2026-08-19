"use client";

interface BrandLogoProps {
  className?: string;
}

export default function BrandLogo({ className = "" }: BrandLogoProps) {
  return (
    <div className={`mx-auto ${className}`}>
      {/* Colored logo — visible in light mode only */}
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img
        src="/predict-a-trade_horizontal.svg"
        alt="Predict-A-Trade"
        className="h-auto dark:hidden"
        style={{ width: "clamp(180px, min(72vw, 30vh), 280px)" }}
      />
      {/* White logo — visible in dark mode only */}
      {/* eslint-disable-next-line @next/next/no-img-element */}
      <img
        src="/predict-a-trade_horizontal_white.svg"
        alt="Predict-A-Trade"
        className="h-auto hidden dark:block"
        style={{ width: "clamp(180px, min(72vw, 30vh), 280px)" }}
      />
    </div>
  );
}
