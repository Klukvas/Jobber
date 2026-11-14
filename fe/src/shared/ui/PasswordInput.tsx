import * as React from "react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Eye, EyeOff } from "lucide-react";
import { cn } from "@/shared/lib/utils";

export type PasswordInputProps = Omit<
  React.InputHTMLAttributes<HTMLInputElement>,
  "type"
>;

const PasswordInput = React.forwardRef<HTMLInputElement, PasswordInputProps>(
  ({ className, ...props }, ref) => {
    const [showPassword, setShowPassword] = useState(false);
    const { t } = useTranslation();

    return (
      <div className="relative">
        <input
          type={showPassword ? "text" : "password"}
          className={cn(
            "flex h-10 max-sm:h-11 w-full rounded-md border border-input bg-background px-3 py-2 pr-10 max-sm:pr-11 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50",
            className,
          )}
          ref={ref}
          {...props}
        />
        <button
          type="button"
          onClick={() => setShowPassword(!showPassword)}
          // The whole right end of the field is the target, not the 16px eye: 40px
          // square on a pointer, 44px on a phone (the field grows to match).
          className="absolute inset-y-0 right-0 flex w-10 max-sm:w-11 items-center justify-center rounded-md text-muted-foreground transition-colors hover:text-foreground focus-visible:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
          aria-label={
            showPassword ? t("auth.hidePassword") : t("auth.showPassword")
          }
        >
          {showPassword ? (
            <EyeOff className="h-4 w-4" />
          ) : (
            <Eye className="h-4 w-4" />
          )}
        </button>
      </div>
    );
  },
);
PasswordInput.displayName = "PasswordInput";

export { PasswordInput };
