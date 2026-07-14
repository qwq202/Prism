import { ThemeProvider } from "@/components/ThemeProviderComponent.tsx";
import DialogManager from "@/dialogs";
import { useEffectAsync } from "@/utils/hook.ts";
import { bindMarket, getApiPlans } from "@/api/v1.ts";
import { useDispatch } from "react-redux";
import {
  stack,
  updateMasks,
  updateSupportModels,
  useMessageActions,
} from "@/store/chat.ts";
import { dispatchSubscriptionData, setTheme } from "@/store/globals.ts";
import { infoEvent } from "@/events/info.ts";
import { setForm } from "@/store/info.ts";
import { themeEvent } from "@/events/theme.ts";
import { useEffect } from "react";
import { getClientCache, setClientCache } from "@/utils/client-cache.ts";
import type { Model, Plan } from "@/api/types.tsx";
import { apiEndpoint } from "@/conf/bootstrap.ts";
import { syncSiteInfo } from "@/admin/api/info.ts";

const marketCacheKey = `market:${apiEndpoint}`;
const plansCacheKey = `plans:${apiEndpoint}`;

function AppProvider({ children }: { children?: React.ReactNode }) {
  const dispatch = useDispatch();
  const { receive } = useMessageActions();

  useEffect(() => {
    infoEvent.bind((data) => dispatch(setForm(data)));
    themeEvent.bind((theme) => dispatch(setTheme(theme)));
    syncSiteInfo();
  }, [dispatch]);

  useEffect(() => {
    stack.setCallback(async (id, message) => {
      await receive(id, message);
    });
  }, [receive]);

  useEffectAsync(async () => {
    const [cachedMarket, cachedPlans] = await Promise.all([
      getClientCache<Model[]>(marketCacheKey),
      getClientCache<Plan[]>(plansCacheKey),
    ]);

    if (cachedMarket?.length) updateSupportModels(dispatch, cachedMarket);
    if (cachedPlans?.length) dispatchSubscriptionData(dispatch, cachedPlans);

    const [market, plans] = await Promise.all([bindMarket(), getApiPlans()]);
    if (market.length) {
      updateSupportModels(dispatch, market);
      void setClientCache(marketCacheKey, market);
    }
    if (plans.length) {
      dispatchSubscriptionData(dispatch, plans);
      void setClientCache(plansCacheKey, plans);
    }

    await updateMasks(dispatch);
  }, []);

  return (
    <ThemeProvider>
      <DialogManager />
      {children}
    </ThemeProvider>
  );
}

export default AppProvider;
