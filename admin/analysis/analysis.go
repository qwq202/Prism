package analysis

import (
	"chat/channel"
	"chat/globals"
	"chat/utils"
	"database/sql"
	"time"

	"github.com/go-redis/redis/v8"
)

type UserTypeForm struct {
	Normal       int64 `json:"normal"`
	ApiPaid      int64 `json:"api_paid"`
	BasicPlan    int64 `json:"basic_plan"`
	StandardPlan int64 `json:"standard_plan"`
	ProPlan      int64 `json:"pro_plan"`
	Total        int64 `json:"total"`
}

func getDates(t []time.Time) []string {
	return utils.Each[time.Time, string](t, func(date time.Time) string {
		return date.Format("1/2")
	})
}

func getFormat(t time.Time) string {
	return t.Format("2006-01-02")
}

func getMinuteFormat(t time.Time) string {
	return t.Format("2006-01-02 15:04")
}

func GetSubscriptionUsers(db *sql.DB) int64 {
	var count int64
	err := globals.QueryRowDb(db, `
   		SELECT COUNT(*) FROM subscription WHERE expired_at > NOW()
   	`).Scan(&count)
	if err != nil {
		return 0
	}

	return count
}

func GetBillingToday(cache *redis.Client) float32 {
	return float32(utils.MustInt(cache, getBillingFormat(getDay()))) / 100
}

func GetBillingYesterday(cache *redis.Client) float32 {
	return float32(utils.MustInt(cache, getBillingFormat(getLastDay()))) / 100
}

func GetBillingMonth(cache *redis.Client) float32 {
	return float32(utils.MustInt(cache, getMonthBillingFormat(getMonth()))) / 100
}

func GetBillingLastMonth(cache *redis.Client) float32 {
	return float32(utils.MustInt(cache, getMonthBillingFormat(getLastMonth()))) / 100
}

func GetTpmToday(cache *redis.Client, user string) int64 {
	// this minute and last minute
	return utils.MustInt(cache, getTpmFormat(getMinuteFormat(time.Now()), user)) +
		utils.MustInt(cache, getTpmFormat(getMinuteFormat(time.Now().Add(-time.Minute)), user))
}

func GetRpmToday(cache *redis.Client, user string) int64 {
	// this minute and last minute
	return utils.MustInt(cache, getRpmFormat(getMinuteFormat(time.Now()), user)) +
		utils.MustInt(cache, getRpmFormat(getMinuteFormat(time.Now().Add(-time.Minute)), user))
}

func GetModelData(cache *redis.Client) ModelChartForm {
	dates := getDays(7)

	return ModelChartForm{
		Date: getDates(dates),
		Value: utils.EachNotNil[string, ModelData](globals.SupportModels, func(model string) *ModelData {
			data := ModelData{
				Model: model,
				Data: utils.Each[time.Time, int64](dates, func(date time.Time) int64 {
					return utils.MustInt(cache, getModelFormat(getFormat(date), model))
				}),
			}
			if utils.Sum(data.Data) == 0 {
				return nil
			}

			return &data
		}),
	}
}

func GetSortedModelData(cache *redis.Client) ModelChartForm {
	form := GetModelData(cache)
	data := utils.Sort(form.Value, func(a ModelData, b ModelData) bool {
		return utils.Sum(a.Data) > utils.Sum(b.Data)
	})

	form.Value = data

	return form
}

func GetRequestData(cache *redis.Client) RequestChartForm {
	dates := getDays(7)

	return RequestChartForm{
		Date: getDates(dates),
		Value: utils.Each[time.Time, int64](dates, func(date time.Time) int64 {
			return utils.MustInt(cache, getRequestFormat(getFormat(date)))
		}),
	}
}

func GetBillingData(cache *redis.Client) BillingChartForm {
	dates := getDays(30)

	return BillingChartForm{
		Date: getDates(dates),
		Value: utils.Each[time.Time, float32](dates, func(date time.Time) float32 {
			return float32(utils.MustInt(cache, getBillingFormat(getFormat(date)))) / 100.
		}),
	}
}

func GetErrorData(cache *redis.Client) ErrorChartForm {
	dates := getDays(7)

	return ErrorChartForm{
		Date: getDates(dates),
		Value: utils.Each[time.Time, int64](dates, func(date time.Time) int64 {
			return utils.MustInt(cache, getErrorFormat(getFormat(date)))
		}),
	}
}

func GetChannelStats(cache *redis.Client, channelIds []int) ChannelStatsResponse {
	today := getDay()
	stats := make([]ChannelStats, 0, len(channelIds))

	for _, id := range channelIds {
		req := utils.MustInt(cache, getChannelRequestFormat(today, id))
		errCount := utils.MustInt(cache, getChannelErrorFormat(today, id))
		total := req + errCount

		var errorRate float64
		if total > 0 {
			errorRate = float64(errCount) / float64(total)
		}

		stats = append(stats, ChannelStats{
			ChannelId: id,
			Requests:  req,
			Errors:    errCount,
			ErrorRate: errorRate,
		})
	}

	return ChannelStatsResponse{Stats: stats}
}

func GetActiveUserData(cache *redis.Client) ActiveUserChartForm {
	dates := getDays(14)

	return ActiveUserChartForm{
		Date: getDates(dates),
		Value: utils.Each[time.Time, int64](dates, func(date time.Time) int64 {
			key := getActiveUserFormat(getFormat(date))
			count, err := cache.SCard(cache.Context(), key).Result()
			if err != nil {
				return 0
			}
			return count
		}),
	}
}

func GetRegistrationData(db *sql.DB) RegistrationChartForm {
	dates := getDays(14)
	dateStrings := getDates(dates)

	counts := make(map[string]int64)
	rows, err := globals.QueryDb(db, `
		SELECT DATE(created_at) as d, COUNT(*) as c
		FROM auth
		WHERE created_at >= ?
		GROUP BY DATE(created_at)
	`, dates[0].Format("2006-01-02"))
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var d string
			var c int64
			if rows.Scan(&d, &c) == nil {
				counts[d] = c
			}
		}
	}

	values := utils.Each[time.Time, int64](dates, func(date time.Time) int64 {
		return counts[getFormat(date)]
	})

	return RegistrationChartForm{
		Date:  dateStrings,
		Value: values,
	}
}

func GetConversionFunnel(db *sql.DB) ConversionFunnelForm {
	var form ConversionFunnelForm

	_ = globals.QueryRowDb(db, `SELECT COUNT(*) FROM auth`).Scan(&form.Registered)
	_ = globals.QueryRowDb(db, `SELECT COUNT(DISTINCT user_id) FROM subscription WHERE total_month > 0`).Scan(&form.EverSubscribed)
	_ = globals.QueryRowDb(db, `SELECT COUNT(*) FROM subscription WHERE expired_at > NOW()`).Scan(&form.ActiveSubscribed)

	return form
}

func GetUserTypeData(db *sql.DB) (UserTypeForm, error) {
	var form UserTypeForm

	// get total users
	if err := globals.QueryRowDb(db, `
		SELECT COUNT(*) FROM auth
	`).Scan(&form.Total); err != nil {
		return form, err
	}

	// get subscription users count (current subscription)
	// level 1: basic plan, level 2: standard plan, level 3: pro plan
	if err := globals.QueryRowDb(db, `
		SELECT
			(SELECT COUNT(*) FROM subscription WHERE level = 1 AND expired_at > NOW()),
			(SELECT COUNT(*) FROM subscription WHERE level = 2 AND expired_at > NOW()),
			(SELECT COUNT(*) FROM subscription WHERE level = 3 AND expired_at > NOW())
	`).Scan(&form.BasicPlan, &form.StandardPlan, &form.ProPlan); err != nil {
		return form, err
	}

	// get normal users count (no subscription in `subscription` table and `quota` + `used` < initial quota in `quota` table)
	initialQuota := channel.SystemInstance.GetInitialQuota()
	if err := globals.QueryRowDb(db, `
		SELECT COUNT(*) FROM auth 
		WHERE id NOT IN (SELECT user_id FROM subscription WHERE total_month > 0)
		AND id IN (SELECT user_id FROM quota WHERE quota + used <= ?)
	`, initialQuota).Scan(&form.Normal); err != nil {
		return form, err
	}

	form.ApiPaid = form.Total - form.Normal - form.BasicPlan - form.StandardPlan - form.ProPlan

	return form, nil
}
