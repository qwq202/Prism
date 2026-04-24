import { useTheme } from "next-themes";
import { Toaster as Sonner } from "sonner";
import React from "react";

type SonnerTheme = "light" | "dark" | "system";
type SonnerPosition =
  | "top-left"
  | "top-right"
  | "bottom-left"
  | "bottom-right"
  | "top-center"
  | "bottom-center";
type SonnerClassnames = {
  toast?: string;
  description?: string;
  actionButton?: string;
  cancelButton?: string;
};
type ToasterProps = React.ComponentProps<typeof Sonner> & {
  theme?: SonnerTheme;
  className?: string;
  toastOptions?: {
    classNames?: SonnerClassnames;
  };
  position?: SonnerPosition;
};

const SonnerToaster = Sonner as React.ComponentType<ToasterProps>;

const Toaster = ({ ...props }: ToasterProps) => {
  const { theme = "system" } = useTheme();

  return (
    <SonnerToaster
      theme={theme as ToasterProps["theme"]}
      className="toaster group"
      toastOptions={{
        classNames: {
          toast:
            "group toast border group-[.toaster]:bg-background group-[.toaster]:text-foreground group-[.toaster]:border-border group-[.toaster]:shadow-lg",
          description: "group-[.toast]:text-muted-foreground",
          actionButton:
            "group-[.toast]:bg-primary group-[.toast]:text-primary-foreground",
          cancelButton:
            "group-[.toast]:bg-muted group-[.toast]:text-muted-foreground",
        },
      }}
      position="top-right"
      // closeButton
      {...props}
    />
  );
};

export { Toaster };
