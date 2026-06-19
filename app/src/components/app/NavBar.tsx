import "@/assets/pages/navbar.less";
import { useTranslation } from "react-i18next";
import { useDispatch, useSelector } from "react-redux";
import {
  selectAuthenticated,
  selectInit,
  selectUsername,
  validateToken,
} from "@/store/auth.ts";
import { Button } from "@/components/ui/button.tsx";
import { Menu, Palette, Settings2 } from "lucide-react";
import { useEffect } from "react";
import { tokenField } from "@/conf/bootstrap.ts";
import { toggleMenu } from "@/store/menu.ts";
import router from "@/router.tsx";
import MenuBar from "./MenuBar.tsx";
import { getMemory } from "@/utils/memory.ts";
import { goAuth } from "@/utils/app.ts";
import Avatar from "@/components/Avatar.tsx";
import { appLogo } from "@/conf/env.ts";
import PrismLogo from "@/components/PrismLogo.tsx";
import { refreshQuota } from "@/store/quota.ts";
import { refreshSubscription } from "@/store/subscription.ts";
import { AppDispatch, clearCronJobs, createCronJob } from "@/store";
import { openDialog } from "@/store/settings.ts";
import { ThemeToggle } from "@/components/ThemeProviderComponent.tsx";
import { infoTimeZoneSelector } from "@/store/info.ts";
import { refreshWalletUsageSummary } from "@/store/record.ts";

function NavMenu() {
  const username = useSelector(selectUsername);

  return (
    <div className={`avatar`}>
      <MenuBar>
        <Button
          variant={`ghost`}
          size={`icon-md`}
          className={`rounded-full overflow-hidden`}
          unClickable
        >
          <Avatar username={username} className={`w-9 h-9 rounded-full`} />
        </Button>
      </MenuBar>
    </div>
  );
}

function NavBar() {
  const { t } = useTranslation();
  const dispatch: AppDispatch = useDispatch();
  useEffect(() => {
    validateToken(dispatch, getMemory(tokenField));
  }, [dispatch]);
  const auth = useSelector(selectAuthenticated);
  const init = useSelector(selectInit);
  const timeZone = useSelector(infoTimeZoneSelector);

  useEffect(() => {
    if (!auth) return;

    const quotaTask = createCronJob(dispatch, refreshQuota, 30, true);
    const planTask = createCronJob(dispatch, refreshSubscription, 30, true);
    const walletStatsTask = createCronJob(
      dispatch,
      () => refreshWalletUsageSummary(timeZone),
      60,
      true,
    );

    return () => clearCronJobs([quotaTask, planTask, walletStatsTask]);
  }, [auth, dispatch, timeZone]);

  return (
    <nav className={`navbar`}>
      <div className={`items space-x-2`}>
        <Button
          size={`icon-md`}
          variant={`ghost`}
          className={`sidebar-button`}
          onClick={() => dispatch(toggleMenu())}
        >
          <Menu className={`w-5 h-5`} />
        </Button>
        {appLogo === "/favicon.svg" ? (
          <PrismLogo
            className={`logo w-9 h-9 scale-110 cursor-pointer`}
            onClick={() => router.navigate("/")}
          />
        ) : (
          <img
            className={`logo w-9 h-9 scale-110`}
            src={appLogo}
            alt=""
            onClick={() => router.navigate("/")}
          />
        )}
        <div className={`grow`} />
        <ThemeToggle size="icon-md" className={`rounded-full overflow-hidden`} />
        <Button
          size={`icon-md`}
          variant={`outline`}
          className={`rounded-full overflow-hidden`}
          onClick={() => dispatch(openDialog())}
        >
          <Settings2 className={`w-4 h-4`} />
        </Button>
        {auth && (
          <Button
            size={`icon-md`}
            variant={`outline`}
            className={`rounded-full overflow-hidden`}
            onClick={() => router.navigate("/drawing")}
            title={t("bar.drawing")}
          >
            <Palette className={`w-4 h-4`} />
          </Button>
        )}
        {!init ? (
          <div className={`h-9 w-24 rounded-full`} />
        ) : auth ? (
          <NavMenu />
        ) : (
          <Button size={`thin`} className={`rounded-full`} onClick={goAuth}>
            {t("login")}
          </Button>
        )}
      </div>
    </nav>
  );
}

export default NavBar;
