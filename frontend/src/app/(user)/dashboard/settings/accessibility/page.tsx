"use client";
import AccessibilitySettings from "@/components/accessibility-settings";

export default function UserAccessibilityPage() {
  return (
    <div className="space-y-4">
      <h1 className="text-xl font-bold">Accessibility Settings</h1>
      <AccessibilitySettings />
    </div>
  );
}
