package katamari

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/cristalhq/base64"

	"github.com/goccy/go-json"

	"github.com/benitogf/katamari/key"
	"github.com/benitogf/katamari/messages"
	"github.com/benitogf/katamari/objects"
	"github.com/stretchr/testify/require"
)

// https://gist.github.com/slaise/9b9d63e0d59e8c8923bbd9d53f5beb61
// https://medium.com/geekculture/my-golang-json-evaluation-20a9ca6ef79c
var TEST_DATA = `{
	"statuses": [
	  {
		"coordinates": null,
		"favorited": false,
		"truncated": false,
		"created_at": "Mon Sep 24 03:35:21 +0000 2012",
		"id_str": "250075927172759552",
		"entities": {
		  "urls": [
  
		  ],
		  "hashtags": [
			{
			  "text": "freebandnames",
			  "indices": [
				20,
				34
			  ]
			}
		  ],
		  "user_mentions": [
  
		  ]
		},
		"in_reply_to_user_id_str": null,
		"contributors": null,
		"text": "Aggressive Ponytail #freebandnames",
		"metadata": {
		  "iso_language_code": "en",
		  "result_type": "recent"
		},
		"retweet_count": 0,
		"in_reply_to_status_id_str": null,
		"id": 250075927172759552,
		"geo": null,
		"retweeted": false,
		"in_reply_to_user_id": null,
		"place": null,
		"user": {
		  "profile_sidebar_fill_color": "DDEEF6",
		  "profile_sidebar_border_color": "C0DEED",
		  "profile_background_tile": false,
		  "name": "Sean Cummings",
		  "profile_image_url": "http://a0.twimg.com/profile_images/2359746665/1v6zfgqo8g0d3mk7ii5s_normal.jpeg",
		  "created_at": "Mon Apr 26 06:01:55 +0000 2010",
		  "location": "LA, CA",
		  "follow_request_sent": null,
		  "profile_link_color": "0084B4",
		  "is_translator": false,
		  "id_str": "137238150",
		  "entities": {
			"url": {
			  "urls": [
				{
				  "expanded_url": null,
				  "url": "",
				  "indices": [
					0,
					0
				  ]
				}
			  ]
			},
			"description": {
			  "urls": [
  
			  ]
			}
		  },
		  "default_profile": true,
		  "contributors_enabled": false,
		  "favourites_count": 0,
		  "url": null,
		  "profile_image_url_https": "https://si0.twimg.com/profile_images/2359746665/1v6zfgqo8g0d3mk7ii5s_normal.jpeg",
		  "utc_offset": -28800,
		  "id": 137238150,
		  "profile_use_background_image": true,
		  "listed_count": 2,
		  "profile_text_color": "333333",
		  "lang": "en",
		  "followers_count": 70,
		  "protected": false,
		  "notifications": null,
		  "profile_background_image_url_https": "https://si0.twimg.com/images/themes/theme1/bg.png",
		  "profile_background_color": "C0DEED",
		  "verified": false,
		  "geo_enabled": true,
		  "time_zone": "Pacific Time (US & Canada)",
		  "description": "Born 330 Live 310",
		  "default_profile_image": false,
		  "profile_background_image_url": "http://a0.twimg.com/images/themes/theme1/bg.png",
		  "statuses_count": 579,
		  "friends_count": 110,
		  "following": null,
		  "show_all_inline_media": false,
		  "screen_name": "sean_cummings"
		},
		"in_reply_to_screen_name": null,
		"source": "<a href=\"//itunes.apple.com/us/app/twitter/id409789998?mt=12%5C%22\" rel=\"\\\"nofollow\\\"\">Twitter for Mac</a>",
		"in_reply_to_status_id": null
	  },
	  {
		"coordinates": null,
		"favorited": false,
		"truncated": false,
		"created_at": "Fri Sep 21 23:40:54 +0000 2012",
		"id_str": "249292149810667520",
		"entities": {
		  "urls": [
  
		  ],
		  "hashtags": [
			{
			  "text": "FreeBandNames",
			  "indices": [
				20,
				34
			  ]
			}
		  ],
		  "user_mentions": [
  
		  ]
		},
		"in_reply_to_user_id_str": null,
		"contributors": null,
		"text": "Thee Namaste Nerdz. #FreeBandNames",
		"metadata": {
		  "iso_language_code": "pl",
		  "result_type": "recent"
		},
		"retweet_count": 0,
		"in_reply_to_status_id_str": null,
		"id": 249292149810667520,
		"geo": null,
		"retweeted": false,
		"in_reply_to_user_id": null,
		"place": null,
		"user": {
		  "profile_sidebar_fill_color": "DDFFCC",
		  "profile_sidebar_border_color": "BDDCAD",
		  "profile_background_tile": true,
		  "name": "Chaz Martenstein",
		  "profile_image_url": "http://a0.twimg.com/profile_images/447958234/Lichtenstein_normal.jpg",
		  "created_at": "Tue Apr 07 19:05:07 +0000 2009",
		  "location": "Durham, NC",
		  "follow_request_sent": null,
		  "profile_link_color": "0084B4",
		  "is_translator": false,
		  "id_str": "29516238",
		  "entities": {
			"url": {
			  "urls": [
				{
				  "expanded_url": null,
				  "url": "http://bullcityrecords.com/wnng/",
				  "indices": [
					0,
					32
				  ]
				}
			  ]
			},
			"description": {
			  "urls": [
  
			  ]
			}
		  },
		  "default_profile": false,
		  "contributors_enabled": false,
		  "favourites_count": 8,
		  "url": "http://bullcityrecords.com/wnng/",
		  "profile_image_url_https": "https://si0.twimg.com/profile_images/447958234/Lichtenstein_normal.jpg",
		  "utc_offset": -18000,
		  "id": 29516238,
		  "profile_use_background_image": true,
		  "listed_count": 118,
		  "profile_text_color": "333333",
		  "lang": "en",
		  "followers_count": 2052,
		  "protected": false,
		  "notifications": null,
		  "profile_background_image_url_https": "https://si0.twimg.com/profile_background_images/9423277/background_tile.bmp",
		  "profile_background_color": "9AE4E8",
		  "verified": false,
		  "geo_enabled": false,
		  "time_zone": "Eastern Time (US & Canada)",
		  "description": "You will come to Durham, North Carolina. I will sell you some records then, here in Durham, North Carolina. Fun will happen.",
		  "default_profile_image": false,
		  "profile_background_image_url": "http://a0.twimg.com/profile_background_images/9423277/background_tile.bmp",
		  "statuses_count": 7579,
		  "friends_count": 348,
		  "following": null,
		  "show_all_inline_media": true,
		  "screen_name": "bullcityrecords"
		},
		"in_reply_to_screen_name": null,
		"source": "web",
		"in_reply_to_status_id": null
	  },
	  {
		"coordinates": null,
		"favorited": false,
		"truncated": false,
		"created_at": "Fri Sep 21 23:30:20 +0000 2012",
		"id_str": "249289491129438208",
		"entities": {
		  "urls": [
  
		  ],
		  "hashtags": [
			{
			  "text": "freebandnames",
			  "indices": [
				29,
				43
			  ]
			}
		  ],
		  "user_mentions": [
  
		  ]
		},
		"in_reply_to_user_id_str": null,
		"contributors": null,
		"text": "Mexican Heaven, Mexican Hell #freebandnames",
		"metadata": {
		  "iso_language_code": "en",
		  "result_type": "recent"
		},
		"retweet_count": 0,
		"in_reply_to_status_id_str": null,
		"id": 249289491129438208,
		"geo": null,
		"retweeted": false,
		"in_reply_to_user_id": null,
		"place": null,
		"user": {
		  "profile_sidebar_fill_color": "99CC33",
		  "profile_sidebar_border_color": "829D5E",
		  "profile_background_tile": false,
		  "name": "Thomas John Wakeman",
		  "profile_image_url": "http://a0.twimg.com/profile_images/2219333930/Froggystyle_normal.png",
		  "created_at": "Tue Sep 01 21:21:35 +0000 2009",
		  "location": "Kingston New York",
		  "follow_request_sent": null,
		  "profile_link_color": "D02B55",
		  "is_translator": false,
		  "id_str": "70789458",
		  "entities": {
			"url": {
			  "urls": [
				{
				  "expanded_url": null,
				  "url": "",
				  "indices": [
					0,
					0
				  ]
				}
			  ]
			},
			"description": {
			  "urls": [
  
			  ]
			}
		  },
		  "default_profile": false,
		  "contributors_enabled": false,
		  "favourites_count": 19,
		  "url": null,
		  "profile_image_url_https": "https://si0.twimg.com/profile_images/2219333930/Froggystyle_normal.png",
		  "utc_offset": -18000,
		  "id": 70789458,
		  "profile_use_background_image": true,
		  "listed_count": 1,
		  "profile_text_color": "3E4415",
		  "lang": "en",
		  "followers_count": 63,
		  "protected": false,
		  "notifications": null,
		  "profile_background_image_url_https": "https://si0.twimg.com/images/themes/theme5/bg.gif",
		  "profile_background_color": "352726",
		  "verified": false,
		  "geo_enabled": false,
		  "time_zone": "Eastern Time (US & Canada)",
		  "description": "Science Fiction Writer, sort of. Likes Superheroes, Mole People, Alt. Timelines.",
		  "default_profile_image": false,
		  "profile_background_image_url": "http://a0.twimg.com/images/themes/theme5/bg.gif",
		  "statuses_count": 1048,
		  "friends_count": 63,
		  "following": null,
		  "show_all_inline_media": false,
		  "screen_name": "MonkiesFist"
		},
		"in_reply_to_screen_name": null,
		"source": "web",
		"in_reply_to_status_id": null
	  },
	  {
		"coordinates": null,
		"favorited": false,
		"truncated": false,
		"created_at": "Fri Sep 21 22:51:18 +0000 2012",
		"id_str": "249279667666817024",
		"entities": {
		  "urls": [
  
		  ],
		  "hashtags": [
			{
			  "text": "freebandnames",
			  "indices": [
				20,
				34
			  ]
			}
		  ],
		  "user_mentions": [
  
		  ]
		},
		"in_reply_to_user_id_str": null,
		"contributors": null,
		"text": "The Foolish Mortals #freebandnames",
		"metadata": {
		  "iso_language_code": "en",
		  "result_type": "recent"
		},
		"retweet_count": 0,
		"in_reply_to_status_id_str": null,
		"id": 249279667666817024,
		"geo": null,
		"retweeted": false,
		"in_reply_to_user_id": null,
		"place": null,
		"user": {
		  "profile_sidebar_fill_color": "BFAC83",
		  "profile_sidebar_border_color": "615A44",
		  "profile_background_tile": true,
		  "name": "Marty Elmer",
		  "profile_image_url": "http://a0.twimg.com/profile_images/1629790393/shrinker_2000_trans_normal.png",
		  "created_at": "Mon May 04 00:05:00 +0000 2009",
		  "location": "Wisconsin, USA",
		  "follow_request_sent": null,
		  "profile_link_color": "3B2A26",
		  "is_translator": false,
		  "id_str": "37539828",
		  "entities": {
			"url": {
			  "urls": [
				{
				  "expanded_url": null,
				  "url": "http://www.omnitarian.me",
				  "indices": [
					0,
					24
				  ]
				}
			  ]
			},
			"description": {
			  "urls": [
  
			  ]
			}
		  },
		  "default_profile": false,
		  "contributors_enabled": false,
		  "favourites_count": 647,
		  "url": "http://www.omnitarian.me",
		  "profile_image_url_https": "https://si0.twimg.com/profile_images/1629790393/shrinker_2000_trans_normal.png",
		  "utc_offset": -21600,
		  "id": 37539828,
		  "profile_use_background_image": true,
		  "listed_count": 52,
		  "profile_text_color": "000000",
		  "lang": "en",
		  "followers_count": 608,
		  "protected": false,
		  "notifications": null,
		  "profile_background_image_url_https": "https://si0.twimg.com/profile_background_images/106455659/rect6056-9.png",
		  "profile_background_color": "EEE3C4",
		  "verified": false,
		  "geo_enabled": false,
		  "time_zone": "Central Time (US & Canada)",
		  "description": "Cartoonist, Illustrator, and T-Shirt connoisseur",
		  "default_profile_image": false,
		  "profile_background_image_url": "http://a0.twimg.com/profile_background_images/106455659/rect6056-9.png",
		  "statuses_count": 3575,
		  "friends_count": 249,
		  "following": null,
		  "show_all_inline_media": true,
		  "screen_name": "Omnitarian"
		},
		"in_reply_to_screen_name": null,
		"source": "<a href=\"//twitter.com/download/iphone%5C%22\" rel=\"\\\"nofollow\\\"\">Twitter for iPhone</a>",
		"in_reply_to_status_id": null
	  }
	],
	"search_metadata": {
	  "max_id": 250126199840518145,
	  "since_id": 24012619984051000,
	  "refresh_url": "?since_id=250126199840518145&q=%23freebandnames&result_type=mixed&include_entities=1",
	  "next_results": "?max_id=249279667666817023&q=%23freebandnames&count=4&include_entities=1&result_type=mixed",
	  "count": 4,
	  "completed_in": 0.035,
	  "since_id_str": "24012619984051000",
	  "query": "%23freebandnames",
	  "max_id_str": "250126199840518145",
	  "something": "something 🧰"
	}
  }`
var units = []string{
	"\xe4\xef\xf0\xe9\xf9l\x100",
	"V'\xe4\xc0\xbb>0\x86j",
	"0'\xe40\x860",
	"\b𝅗𝅝\x85",
	"𓏝",
	"𝅅",
	"'",
	"\xd80''",
	"\xd8%''",
	"0",
	"",
}

// StorageObjectTest testing storage function
func StorageObjectTest(app *Server, t *testing.T) {
	app.Storage.Clear()
	index, err := app.Storage.Set("test", "test")
	require.NoError(t, err)
	require.NotEmpty(t, index)
	data, _ := app.Storage.Get("test")
	testObject, err := objects.DecodeRaw(data)
	require.NoError(t, err)
	require.Equal(t, "test", testObject.Data)
	require.Equal(t, int64(0), testObject.Updated)
	index, err = app.Storage.Set("test", "test_update")
	require.NoError(t, err)
	require.NotEmpty(t, index)
	data, err = app.Storage.Get("test")
	require.NoError(t, err)
	testObject, err = objects.DecodeRaw(data)
	require.NoError(t, err)
	require.Equal(t, "test_update", testObject.Data)
	err = app.Storage.Del("test")
	require.NoError(t, err)
	raw, _ := app.Storage.Get("test")
	dataDel := string(raw)
	require.Empty(t, dataDel)
}

// StorageListTest testing storage function
func StorageListTest(app *Server, t *testing.T, testData string) {
	app.Storage.Clear()
	modData := testData + testData
	key, err := app.Storage.Set("test/123", testData)
	require.NoError(t, err)
	require.Equal(t, "123", key)
	key, err = app.Storage.Set("test/456", modData)
	require.NoError(t, err)
	require.Equal(t, "456", key)
	data, err := app.Storage.Get("test/*")
	require.NoError(t, err)
	var testObjects []objects.Object
	err = json.Unmarshal(data, &testObjects)
	require.NoError(t, err)
	require.Equal(t, 2, len(testObjects))
	for i := range testObjects {
		if testObjects[i].Index == "123" {
			require.Equal(t, testData, testObjects[i].Data)
		}

		if testObjects[i].Index == "456" {
			require.Equal(t, modData, testObjects[i].Data)
		}
	}
	data1, err := app.Storage.Get("test/123")
	require.NoError(t, err)
	data2, err := app.Storage.Get("test/456")
	require.NoError(t, err)
	obj1, err := objects.DecodeRaw(data1)
	require.NoError(t, err)
	obj2, err := objects.DecodeRaw(data2)
	require.NoError(t, err)
	require.Equal(t, testData, obj1.Data)
	require.Equal(t, modData, obj2.Data)
	keys, err := app.Storage.Keys()
	require.NoError(t, err)
	require.Equal(t, "{\"keys\":[\"test/123\",\"test/456\"]}", string(keys))

	req := httptest.NewRequest(
		"POST", "/test/*",
		bytes.NewBuffer(
			[]byte(`{"data":"testpost"}`),
		),
	)
	w := httptest.NewRecorder()
	app.Router.ServeHTTP(w, req)
	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	dat, err := objects.DecodeRaw(body)
	require.NoError(t, err)
	data, err = app.Storage.Get("test/*")
	app.Console.Log(string(data))
	require.NoError(t, err)
	err = json.Unmarshal(data, &testObjects)
	require.NoError(t, err)
	require.Equal(t, 3, len(testObjects))
	err = app.Storage.Del("test/" + dat.Index)
	require.NoError(t, err)
	data, err = app.Storage.Get("test/*")
	require.NoError(t, err)
	err = json.Unmarshal(data, &testObjects)
	require.NoError(t, err)
	require.Equal(t, 2, len(testObjects))
	key, err = app.Storage.Set("test/glob1/glob123", testData)
	require.NoError(t, err)
	require.Equal(t, "glob123", key)
	key, err = app.Storage.Set("test/glob2/glob456", modData)
	require.NoError(t, err)
	require.Equal(t, "glob456", key)
	data, err = app.Storage.Get("test/*/*")
	require.NoError(t, err)
	err = json.Unmarshal(data, &testObjects)
	app.Console.Log(testObjects)
	require.NoError(t, err)
	require.Equal(t, 2, len(testObjects))
	key, err = app.Storage.Set("test/1/glob/g123", testData)
	require.NoError(t, err)
	require.Equal(t, "g123", key)
	key, err = app.Storage.Set("test/2/glob/g456", modData)
	require.NoError(t, err)
	require.Equal(t, "g456", key)
	data, err = app.Storage.Get("test/*/glob/*")
	require.NoError(t, err)
	err = json.Unmarshal(data, &testObjects)
	app.Console.Log(testObjects)
	require.NoError(t, err)
	require.Equal(t, 2, len(testObjects))
	key, err = app.Storage.Set("test1", testData)
	require.NoError(t, err)
	require.Equal(t, "test1", key)
	key, err = app.Storage.Set("test2", modData)
	require.NoError(t, err)
	require.Equal(t, "test2", key)
	data, err = app.Storage.Get("*")
	require.NoError(t, err)
	err = json.Unmarshal(data, &testObjects)
	app.Console.Log(testObjects)
	require.NoError(t, err)
	require.Equal(t, 2, len(testObjects))
	err = app.Storage.Del("*")
	require.NoError(t, err)
	data, err = app.Storage.Get("*")
	require.NoError(t, err)
	err = json.Unmarshal(data, &testObjects)
	app.Console.Log(testObjects)
	require.NoError(t, err)
	require.Equal(t, 0, len(testObjects))
}

// StorageSetGetDelTest testing storage function
func StorageSetGetDelTest(db Database, b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testData := messages.Encode([]byte(TEST_DATA))
		_key := key.Build("test/*")
		_, err := db.Set("test/"+_key, testData)
		require.NoError(b, err)
		fetched, err := db.Get("test/" + _key)
		require.NoError(b, err)
		decoded, err := objects.Decode(fetched)
		require.NoError(b, err)
		require.Equal(b, decoded.Data, TEST_DATA)
		err = db.Del("test/" + _key)
		require.NoError(b, err)
		result, err := db.Get("test/*")
		require.NoError(b, err)
		require.Equal(b, "[]", string(result))
	}
}

// StorageGetNTest testing storage GetN function
func StorageGetNTest(app *Server, t *testing.T, n int) {
	app.Storage.Clear()
	testData := base64.StdEncoding.EncodeToString([]byte(TEST_DATA))
	for i := 0; i < n; i++ {
		value := strconv.Itoa(i)
		key, err := app.Storage.Set("test/"+value, testData)
		require.NoError(t, err)
		require.Equal(t, value, key)
		time.Sleep(time.Millisecond * 1)
	}

	limit := 1
	testObjects, err := app.Storage.GetN("test/*", limit)
	require.NoError(t, err)
	require.Equal(t, limit, len(testObjects))
	require.Equal(t, strconv.Itoa(n-1), testObjects[0].Index)
}

// StorageGetNRangeTest testing storage GetN function
func StorageGetNRangeTest(app *Server, t *testing.T, n int) {
	app.Storage.Clear()
	testData := base64.StdEncoding.EncodeToString([]byte(TEST_DATA))
	for i := 1; i < n; i++ {
		value := strconv.Itoa(i)
		key, err := app.Storage.Pivot("test/"+value, testData, int64(i), 0)
		require.NoError(t, err)
		require.Equal(t, value, key)
		time.Sleep(time.Millisecond * 1)
	}

	_, err := app.Storage.Pivot("test/0", testData, 0, 0)
	require.NoError(t, err)

	limit := 1
	testObjects, err := app.Storage.GetNRange("test/*", limit, 0, 1)
	require.NoError(t, err)
	require.Equal(t, limit, len(testObjects))
	// require.Equal(t, int64(0), testObjects[0].Created)
	// require.Equal(t, "0", testObjects[0].Index)
	require.Equal(t, int64(1), testObjects[0].Created)
	require.Equal(t, "1", testObjects[0].Index)
}

// StorageKeysRangeTest testing storage GetN function
func StorageKeysRangeTest(app *Server, t *testing.T, n int) {
	app.Storage.Clear()
	testData := TEST_DATA
	first := ""
	for i := 0; i < n; i++ {
		path := key.Build("test/*")
		key, err := app.Storage.Set(path, testData)
		if first == "" {
			first = key
		}
		require.NoError(t, err)
		require.Equal(t, path, "test/"+key)
		if runtime.GOOS == "windows" {
			// time granularity in windows is not fast enough
			time.Sleep(time.Millisecond * 1)
		}
	}

	keys, err := app.Storage.KeysRange("test/*", 0, key.Decode(first))
	require.NoError(t, err)
	require.Equal(t, 1, len(keys))
	require.Equal(t, "test/"+first, keys[0])
}
