import { useTranslation } from "react-i18next";

import { Input } from "@/shared/ui/Input";
import { Label } from "@/shared/ui/Label";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/shared/ui/Card";
import { useProfile } from "./useProfile";

/**
 * Shows the account Jobber has on file, read from the server rather than from
 * the copy the browser cached at sign-in.
 *
 * Nothing here writes. The email is a login credential and changing it needs a
 * verification round-trip; the display name has no write endpoint either. What
 * this card is for is being *true*: the auth store is written once at sign-in
 * and never refreshed, so a change made anywhere else would otherwise show
 * here as whatever was current months ago.
 */
export function ProfileSection() {
  const { t } = useTranslation();
  const { user, isError } = useProfile();

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("settings.profile.title")}</CardTitle>
        <CardDescription>{t("settings.profile.description")}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="profile-name">{t("settings.profile.name")}</Label>
            <Input
              id="profile-name"
              value={user?.name ?? ""}
              readOnly
              disabled
              aria-describedby="profile-name-hint"
            />
            <p id="profile-name-hint" className="text-xs text-muted-foreground">
              {t("settings.profile.nameHint")}
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="profile-email">{t("settings.profile.email")}</Label>
            <Input
              id="profile-email"
              value={user?.email ?? ""}
              readOnly
              disabled
              aria-describedby="profile-email-hint"
            />
            <p id="profile-email-hint" className="text-xs text-muted-foreground">
              {t("settings.profile.emailReadOnly")}
            </p>
          </div>

          {/* A failed refresh is worth saying out loud — what is on screen is
              the last known account, not nothing — but it must not read as an
              error the customer caused. */}
          {isError && (
            <p role="status" className="text-xs text-muted-foreground">
              {t("settings.profile.refreshFailed")}
            </p>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
