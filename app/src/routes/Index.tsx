import { Outlet, useLocation } from "react-router-dom";
import ErrorBoundary from "@/components/ErrorBoundary.tsx";
import "@/assets/pages/home.less";
import { Button } from "@/components/ui/button.tsx";
import {
  ChevronDown,
  MessageCircle,
  PieChart,
  Shield,
  Sparkles,
  Wallet,
  LibraryBig,
  User,
} from "lucide-react";
import React from "react";
import Icon from "@/components/utils/Icon.tsx";
import router from "@/router.tsx";
import { useTranslation } from "react-i18next";
import { cn } from "@/components/ui/lib/utils.ts";
import { useSelector } from "react-redux";
import { selectAdmin } from "@/store/auth.ts";
import {
  hideToolbarSelector,
  hideToolbarTextSelector,
} from "@/store/settings.ts";
import { isMobile, useMobile } from "@/utils/device.ts";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip.tsx";
import NavBar from "@/components/app/NavBar.tsx";

type BarItemProps = {
  icon: React.ReactElement;
  path: string;
  name: string;
};

function isPrefix(current: string, path: string): boolean {
  if (location.pathname === path) return true;
  if (location.pathname + "/" === path) return true;

  return path.length > 1 && current.startsWith(path + "/");
}

function BarItem({ icon, path, name }: BarItemProps) {
  const { t } = useTranslation();
  const location = useLocation();
  const active = isPrefix(location.pathname, path);

  const hidden = useSelector(hideToolbarTextSelector);
  const mobile = useMobile();

  const [open, setOpen] = React.useState(false);

  const onClick = async () => {
    await router.navigate(path);
  };

  return (
    <div className={`inline-flex flex-col`}>
      <TooltipProvider delayDuration={100}>
        <Tooltip open={open} onOpenChange={setOpen}>
          <TooltipTrigger asChild>
            <Button
              size="icon"
              variant={active ? "default" : "outline"}
              onClick={onClick}
            >
              <Icon icon={icon} className="h-4 w-4 stroke-[1.75]" />
            </Button>
          </TooltipTrigger>
          <TooltipContent
            side={mobile ? "top" : "right"}
            align="center"
            className={`z-[100]`}
          >
            {t(`bar.${name}`)}
          </TooltipContent>
        </Tooltip>
      </TooltipProvider>
      <div
        className={cn(
          `toolbar-text text-secondary text-center text-xs mt-1.5 cursor-pointer select-none`,
          active && `text-common`,
          hidden && `hidden`,
        )}
        onClick={onClick}
      >
        {t(`bar.${name}`)}
      </div>
    </div>
  );
}

function ToolBar() {
  const admin = useSelector(selectAdmin);
  const hideToolbar = useSelector(hideToolbarSelector);
  const [stacked, setStacked] = React.useState(hideToolbar || isMobile());

  return (
    <div className={cn("toolbar", stacked && "stacked")}>
      <div
        className={cn("bar-kit", stacked && "stacked")}
        onClick={(e) => {
          e.stopPropagation();
          e.preventDefault();
          setStacked(!stacked);
        }}
      >
        <ChevronDown className={`h-3.5 w-3.5`} />
      </div>
      <BarItem icon={<MessageCircle />} path={`/`} name={"chat"} />
      <BarItem
        icon={<Sparkles />}
        path={`/personalization`}
        name={"personalization"}
      />
      <BarItem icon={<LibraryBig />} path={`/model`} name={"model"} />
      {/* <BarItem icon={<Compass />} path={`/preset`} name={"preset"} /> */}
      <BarItem icon={<Wallet />} path={`/wallet`} name={"wallet"} />
      {/* <BarItem icon={<DraftingCompass />} path={`/key`} name={"key"} /> */}
      <BarItem icon={<PieChart />} path={`/log`} name={"log"} />
      <BarItem icon={<User />} path={`/account`} name={"account"} />
      {admin && <BarItem icon={<Shield />} path={`/admin`} name={"admin"} />}
    </div>
  );
}

function Home() {
  return (
    <ErrorBoundary>
      <NavBar />
      <div className={`main relative`}>
        <ToolBar />
        <Outlet />
      </div>
    </ErrorBoundary>
  );
}

export default Home;
