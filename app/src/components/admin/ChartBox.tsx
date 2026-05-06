import ModelChart from "@/components/admin/assemblies/ModelChart.tsx";
import { useState } from "react";
import {
  ActiveUserChartResponse,
  BillingChartResponse,
  ConversionFunnelResponse,
  ErrorChartResponse,
  ModelChartResponse,
  RegistrationChartResponse,
  RequestChartResponse,
  UserTypeChartResponse,
} from "@/admin/types.ts";
import RequestChart from "@/components/admin/assemblies/RequestChart.tsx";
import BillingChart from "@/components/admin/assemblies/BillingChart.tsx";
import ErrorChart from "@/components/admin/assemblies/ErrorChart.tsx";
import { useEffectAsync } from "@/utils/hook.ts";
import {
  getActiveUserChart,
  getBillingChart,
  getConversionFunnel,
  getErrorChart,
  getModelChart,
  getRegistrationChart,
  getRequestChart,
  getUserTypeChart,
} from "@/admin/api/chart.ts";
import ModelUsageChart from "@/components/admin/assemblies/ModelUsageChart.tsx";
import UserTypeChart from "@/components/admin/assemblies/UserTypeChart.tsx";
import ChannelHealthChart from "@/components/admin/assemblies/ChannelHealthChart.tsx";
import { getChannelStats, type ChannelStat } from "@/admin/api/channel.ts";
import { listChannel } from "@/admin/api/channel.ts";
import { Channel } from "@/admin/channel.ts";
import ActiveUserChart from "@/components/admin/assemblies/ActiveUserChart.tsx";
import RegistrationChart from "@/components/admin/assemblies/RegistrationChart.tsx";
import ConversionFunnelChart from "@/components/admin/assemblies/ConversionFunnelChart.tsx";

function ChartBox() {
  const [model, setModel] = useState<ModelChartResponse>({
    date: [],
    value: [],
  });

  const [request, setRequest] = useState<RequestChartResponse>({
    date: [],
    value: [],
  });

  const [billing, setBilling] = useState<BillingChartResponse>({
    date: [],
    value: [],
  });

  const [error, setError] = useState<ErrorChartResponse>({
    date: [],
    value: [],
  });

  const [user, setUser] = useState<UserTypeChartResponse>({
    total: 0,
    normal: 0,
    api_paid: 0,
    basic_plan: 0,
    standard_plan: 0,
    pro_plan: 0,
  });

  const [channelStats, setChannelStats] = useState<ChannelStat[]>([]);
  const [channels, setChannels] = useState<Channel[]>([]);

  const [activeUser, setActiveUser] = useState<ActiveUserChartResponse>({
    date: [],
    value: [],
  });

  const [registration, setRegistration] = useState<RegistrationChartResponse>({
    date: [],
    value: [],
  });

  const [funnel, setFunnel] = useState<ConversionFunnelResponse>({
    registered: 0,
    ever_subscribed: 0,
    active_subscribed: 0,
  });

  useEffectAsync(async () => {
    setModel(await getModelChart());
    setRequest(await getRequestChart());
    setBilling(await getBillingChart());
    setError(await getErrorChart());
    setUser(await getUserTypeChart());
    const [statsResp, channelResp] = await Promise.all([
      getChannelStats(),
      listChannel(),
    ]);
    setChannelStats(statsResp.stats ?? []);
    if (channelResp.status) setChannels(channelResp.data);
    setActiveUser(await getActiveUserChart());
    setRegistration(await getRegistrationChart());
    setFunnel(await getConversionFunnel());
  }, []);

  return (
    <div className={`chart-boxes`}>
      <div className={`chart-box`}>
        <ModelChart labels={model.date} datasets={model.value} />
      </div>
      <div className={`chart-box`}>
        <ModelUsageChart labels={model.date} datasets={model.value} />
      </div>
      <div className={`chart-box`}>
        <BillingChart labels={billing.date} datasets={billing.value} />
      </div>
      <div className={`chart-box`}>
        <UserTypeChart data={user} />
      </div>
      <div className={`chart-box`}>
        <RequestChart labels={request.date} datasets={request.value} />
      </div>
      <div className={`chart-box`}>
        <ErrorChart labels={error.date} datasets={error.value} />
      </div>
      <div className={`chart-box`}>
        <ChannelHealthChart stats={channelStats} channels={channels} />
      </div>
      <div className={`chart-box`}>
        <ActiveUserChart labels={activeUser.date} datasets={activeUser.value} />
      </div>
      <div className={`chart-box`}>
        <RegistrationChart labels={registration.date} datasets={registration.value} />
      </div>
      <div className={`chart-box`}>
        <ConversionFunnelChart data={funnel} />
      </div>
    </div>
  );
}

export default ChartBox;
