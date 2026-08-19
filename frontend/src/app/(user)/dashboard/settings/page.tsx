"use client";
import AccessibilitySettings from "@/components/accessibility-settings";

export default function UserSettingsPage() {
  return (
    <div className="space-y-4">
      <h1 className="text-xl font-bold">Settings</h1>
      <AccessibilitySettings />
    </div>
  );
}
