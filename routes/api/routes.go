package api

import (
	"context"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"

	"hyperpage/controllers"
	"hyperpage/controllers/crypto"

	"hyperpage/initializers"
	"hyperpage/middleware"
)

func Register(micro *fiber.App) {

	micro.Route("/orders", func(router fiber.Router) {
		router.Post("/newOrder", middleware.DeserializeUser, controllers.CreateOrder)
		router.Get("/getOrders",  middleware.DeserializeUser, controllers.GetOrdersForSeller)
		router.Get("/getOrdersBuyers",  middleware.DeserializeUser, controllers.GetOrdersForBuyer)
				
		router.Post("/addAddr",  middleware.DeserializeUser, controllers.AddDeliveryAddress)
		router.Delete("/delAddr/:id", middleware.DeserializeUser, controllers.DeleteDeliveryAddress) 
		router.Put("/editAddr/:id", middleware.DeserializeUser, controllers.UpdateDeliveryAddress)  
        router.Get("/getAddresses", middleware.DeserializeUser, controllers.GetDeliveryAddresses)    
		router.Patch("/:id/status", middleware.DeserializeUser, controllers.UpdateOrderStatus)

	})

	micro.Route("/post", func(router fiber.Router) {
		router.Get("/get/:id", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.GetPostByID)
		router.Get("/feed", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.GetUserAndFollowingsPosts)

		router.Post("/create", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.CreatePost)
		router.Get("/get", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.GetUserPosts)
		router.Delete("/delete/:id", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.DeletePost)
		router.Patch("/update/:id", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.UpdatePost)

		router.Post("/:id/likes", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.ToggleLike)

		router.Post("/:id/comments", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.AddComment)
		router.Get("/:id/comments", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.GetComments)
		router.Delete("/:id/comments/:commentId", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.DeleteComment)

	})

	micro.Route("/settings", func(router fiber.Router) {
		router.Get("/youtube/*", controllers.ProxyYouTube)
		router.Get("/base", controllers.GetBaseSystemData)
		router.Get("/langs", controllers.Langs)
		router.Post("/addlang", middleware.DeserializeUser, middleware.CheckRole([]string{"admin"}), controllers.AddLang)
		router.Delete("/deletelang/:id", middleware.DeserializeUser, middleware.CheckRole([]string{"admin"}), controllers.DeleteLang)
		router.Patch("/updatelang/:id", middleware.DeserializeUser, middleware.CheckRole([]string{"admin"}), controllers.UpdateLang)
	})

	micro.Route("/presavedfilter", func(router fiber.Router) {
		router.Get("/get", middleware.DeserializeUser, controllers.GetPresavedfilters)
		router.Post("/post", middleware.DeserializeUser, controllers.CreatePresavedfilter)
		router.Patch("/patch/:id", middleware.DeserializeUser, controllers.PatchPresavedFilter)
		router.Delete("/delete/:id", middleware.DeserializeUser, controllers.DeletePresavedFilter)
	})

	micro.Route("/devices", func(router fiber.Router) {
		router.Post("/ios", controllers.CreateDevice)
		router.Post("/push", controllers.SendNot)
	})

	micro.Route("/relations", func(router fiber.Router) {
		router.Get("/following", middleware.DeserializeUser, controllers.GetFollowing)
		router.Get("/followers", middleware.DeserializeUser, controllers.GetFollowers)
		router.Post("/send-push", middleware.DeserializeUser, controllers.SendPushNotification)
	})

	micro.Route("/crypto", func(router fiber.Router) {
		router.Get("/balance", middleware.DeserializeUser, crypto.Balance)
		router.Post("/walllet", middleware.DeserializeUser, crypto.Wallet)
		router.Get("/transactions", middleware.DeserializeUser, crypto.GetTransactions)
		router.Post("/walllet/send", middleware.DeserializeUser, crypto.SendCoins)
	})


	micro.Route("/newreq", func(router fiber.Router) {
		router.Post("/post", controllers.Userq)
	})

	micro.Route("/auth", func(router fiber.Router) {
		router.Post("/register", controllers.SignUpUser)
		router.Post("/login", controllers.SignInUser)
		router.Post("/forgotpassword", controllers.ForgotPassword)
		router.Patch("/resetpassword/:resetToken", controllers.ResetPassword)
		router.Get("/verifyemail/:verificationCode", controllers.VerifyEmail)
		router.Get("/logout", middleware.DeserializeUser, controllers.LogoutUser)
		router.Get("/refresh/:refreshToken", controllers.RefreshAccessToken)
		router.Post("/checkTokenExp", controllers.CheckTokenExp)
		router.Get("/check", middleware.DeserializeUser, controllers.GetUserDetails)
	})

	micro.Route("/followers", func(router fiber.Router) {
		router.Post("/scribe", middleware.DeserializeUser, controllers.Scribe)
		router.Post("/unscribe", middleware.DeserializeUser, controllers.Unscribe)
		router.Get("/get", middleware.DeserializeUser, controllers.GetFollowers)
	})

	micro.Route("/domains", func(router fiber.Router) {
		router.Get("/get", controllers.GetDomain)
	})

	micro.Route("/site", func(router fiber.Router) {
		router.Post("/update", middleware.DeserializeUser, controllers.UpdateSite)
		router.Get("/get", middleware.DeserializeUser, controllers.GetSite)
	})

	micro.Route("/users", func(router fiber.Router) {
		router.Get("/myTime", controllers.MyTime)
		router.Post("/deletme", middleware.DeserializeUser, controllers.DeleteUserWithRelations)
		router.Post("/setvip", middleware.DeserializeUser, controllers.SetVipUser)
		router.Patch("/changeName", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.ChangeNickName)
		router.Patch("/setTokenDeivce", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.SetTokenIOSdevice)
		router.Get("/notifications", middleware.DeserializeUser, controllers.GetNotifications)
		router.Patch("/notifications/:id/read", middleware.DeserializeUser, controllers.MarkNotificationAsRead)
		router.Delete("/notifications/:id", middleware.DeserializeUser, controllers.DeleteNotification)
		router.Put("/changePhoto", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.ChangePhoto)


		router.Post("/sendrequestcall", controllers.SendBotCallRequest)
		// router.Get("/me", middleware.DeserializeUser, controllers.GetMe)
		router.Get("/me", func(c *fiber.Ctx) error {
			// Capture the language from the URL, headers, or any other source.
			language := c.Query("language") // Example: ?language=en
			if language == "" {
				language = "en"
			}
			// Set the language in the context for middleware.
			c.Locals("language", language)

			// Call the DeserializeUser middleware.
			return middleware.DeserializeUser(c)
		}, controllers.GetMe)
		router.Get("/getmefirst", middleware.DeserializeUser, controllers.GetMeFirst)
		router.Post("/addbalance", middleware.DeserializeUser, controllers.AddBalance)
		router.Post("/plan", middleware.DeserializeUser, controllers.Plan)
	})

	micro.Route("/billing", func(router fiber.Router) {
		router.Get("/transactions", middleware.DeserializeUser, controllers.GetTransactions)
	})

	micro.Route("/calls", func(router fiber.Router) {
		router.Post("/makecall", controllers.MakeCall)
		router.Post("/stopcall", controllers.StopCall)
	})

	micro.Route("/cities", func(router fiber.Router) {
		router.Get("/all", controllers.GetCities)
		router.Get("/query", controllers.GetName)
		router.Post("/create", middleware.DeserializeUser, middleware.CheckRole([]string{"admin"}), controllers.CreateCity)
		router.Delete("/remove/:id", middleware.DeserializeUser, middleware.CheckRole([]string{"admin"}), controllers.DeleteCity)
		router.Patch("/update/:id", middleware.DeserializeUser, middleware.CheckRole([]string{"admin"}), controllers.UpdateCity)
		router.Get("/get", middleware.DeserializeUser, middleware.CheckRole([]string{"admin"}), controllers.GetCityTranslation)
	})

	micro.Route("/citiestranslator", func(router fiber.Router) {
		router.Post("/create", middleware.DeserializeUser, middleware.CheckRole([]string{"admin"}), controllers.CreateCityTranslation)
		router.Delete("/remove", middleware.DeserializeUser, middleware.CheckRole([]string{"admin"}), controllers.DeleteCityTranslation)
		router.Patch("/update", middleware.DeserializeUser, middleware.CheckRole([]string{"admin"}), controllers.UpdateCityTranslation)
	})

	micro.Route("/guilds", func(router fiber.Router) {
		router.Get("/all", controllers.GetGuilds)
		router.Get("/getAll", controllers.GetGuildsAll)
		router.Post("/create", middleware.DeserializeUser, middleware.CheckRole([]string{"admin"}), controllers.CreateGuild)
		router.Delete("/remove/:id", middleware.DeserializeUser, middleware.CheckRole([]string{"admin"}), controllers.DeleteGuild)
		router.Patch("/update/:id", middleware.DeserializeUser, middleware.CheckRole([]string{"admin"}), controllers.UpdateGuild)

		router.Get("/name", controllers.GetGuildName)
		router.Get("/namecustom", controllers.GetGuildNameA)
	})

	micro.Route("/guildstranslator", func(router fiber.Router) {
		router.Post("/create", middleware.DeserializeUser, middleware.CheckRole([]string{"admin"}), controllers.CreateGuildTranslation)
		router.Delete("/remove", middleware.DeserializeUser, middleware.CheckRole([]string{"admin"}), controllers.DeleteGuildTranslation)
		router.Patch("/update", middleware.DeserializeUser, middleware.CheckRole([]string{"admin"}), controllers.UpdateGuildTranslation)
	})

	micro.Route("/profile", func(router fiber.Router) {
		router.Get("/get", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.GetProfile)
		router.Patch("/save", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.UpdateProfile)
		router.Patch("/saveAdditional", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.UpdateProfileAdditional)
		router.Patch("/photos", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.UpdateProfilePhotos)
		router.Post("/documents", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.NewProfileDocuments)
		router.Patch("/documents", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.UpdateProfileDocuments)
		router.Delete("/documents/:id", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.DeleteProfileDocuments)
		router.Post("/streaming/", controllers.UpdateProfileStreaming)
		router.Delete("/streaming/:id", controllers.DeleteProfileStreaming)
		router.Post("/streaming/donat", middleware.DeserializeUser, controllers.SendDonat)

		router.Get("/getdocuments", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.GetDocuments)
	})

	micro.Route("/profiles", func(router fiber.Router) {
		router.Get("/get", controllers.GetAllProfile)
		router.Get("/get/:name", controllers.GetProfileGuest)
		router.Get("/streaming", controllers.GetProfiles)
	})

	micro.Route("/payment", func(router fiber.Router) {
		router.Post("/invoice", middleware.DeserializeUser, controllers.CreateInvoice)
		router.Post("/pending", controllers.Pending)
	})

	micro.Route("/profilehashtags", func(router fiber.Router) {
		router.Post("/addhashtag", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.AddHashTagProfile)
		router.Get("/findTag", controllers.SearchHashTagProfile)
		router.Get("/get", controllers.Get10RandomTags)

	})

	micro.Route("/blog", func(router fiber.Router) {
		router.Get("/list", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.GetAllBlogs)
		router.Post("/makearchive/:id", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.SendToArchive)
		router.Post("/search", middleware.DeserializeUser, controllers.SearchBlogByTitle)
		router.Post("/addblogtime", middleware.DeserializeUser, controllers.AddBlogTime)
		router.Post("/addhashtag", middleware.DeserializeUser, controllers.AddHashTag)
		router.Get("/findTag", controllers.SearchHashTag)
		router.Get("/taketags", controllers.Get10RandomBlogHashtags)
		router.Post("/filterByIds", controllers.FilterBlogsWithIds)

		router.Post("/addFav", middleware.DeserializeUser, controllers.AddFav)
		router.Get("/getFav", middleware.DeserializeUser, controllers.GetFavorites)
		router.Delete("/delFav", middleware.DeserializeUser, controllers.DelFav)

		router.Get("/allvotes/:id", controllers.GetAllVotes)
		router.Post("/addvote/:id", middleware.DeserializeUser, controllers.AddVote)

		router.Get("/getAllByUser/:id", controllers.GetAllByUser)

		router.Get("/listAll", controllers.GetAll)

		router.Get("/random", controllers.GetRandom)

		router.Get("/:id", controllers.GetBlogById)
		router.Post("/create", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), middleware.CheckProfileFilled(), controllers.CreateBlog)
		router.Post("/create/photos", middleware.DeserializeUser, controllers.CreateBlogPhoto)
		router.Get("/edit/:id", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.EditBlogGetId)
		router.Patch("/patch/:id", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.UpdateBlog)
		router.Delete("/delete/:id", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.DeleteBlog)
	})

	micro.Route("/chat", func(router fiber.Router) {
		router.Get("/room/:roomId", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.GetRoomDetailsForDM)
		router.Get("/rooms", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.GetSubscribedRoomsForDM)
		router.Get("/newRooms", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.GetNewUnsubscribedRoomsForDM)
		router.Get("/archivedRooms", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.GetUnsubscribedNotNewRoomsForDM)
		router.Post("/createRoom", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.CreateChatRoomForDM)
		router.Patch("/subscribe/:roomId", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.SubscribeNewRoomForDM)
		router.Patch("/unsubscribe/:roomId", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.UnsubscribeRoomForDM)

		router.Get("/message/:roomId", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.GetChatMessagesForDM)
		router.Post("/message/:roomId", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.SendMessageForDM)
		router.Patch("/message/:messageId", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.EditMessageForDM)
		router.Delete("/message/:messageId", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.DeleteMessageForDM)
		// Marks a message as read by the recipient
		router.Patch("/read/:roomId", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.MarkMessageAsReadForDM)
		router.Patch("/unread/:roomId/:status", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.MarkMessageAsUnReadForDM)
	})

	micro.Route("/contrifugoToken", func(router fiber.Router) {
		router.Get("/connection", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.GetCentrifugoConnectionToken)
		router.Get("/subscription", middleware.DeserializeUser, middleware.CheckRole([]string{"admin", "user", "vip"}), controllers.GetCentrifugoSubscriptionToken)
	})

	micro.Route("/files", func(router fiber.Router) {
		router.Post("/upload/file", middleware.DeserializeUser, middleware.CheckProfileFilled(), controllers.UploadPdf)
		router.Post("/upload", middleware.DeserializeUser, middleware.CheckProfileFilled(), controllers.UploadImage)
		router.Post("/upload/images", middleware.DeserializeUser, middleware.CheckProfileFilled(), controllers.UploadImages)
	})

	micro.Route("/server", func(router fiber.Router) {
		ctx := context.TODO()
		value, err := initializers.RedisClient.Get(ctx, "statusHealth").Result()

		router.Get("/healthchecker", func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"status":  "success",
				"message": value,
			})
		})

		if err == redis.Nil {
			fmt.Println("key: statusHealth does not exist")
		} else if err != nil {
			panic(err)
		}
	})

	micro.Route("/managebot", func(router fiber.Router) {
		router.Post("/registerbot", controllers.SignUpBot)
		router.Post("/deletebots", controllers.DeleteAllBotUsersWithRelations)
		router.Patch("/updateprofile", controllers.UpdateBotProfile)
		router.Patch("/updateadditionalinfo", controllers.UpdateBotProfileAdditional)
	})

	micro.All("*", func(c *fiber.Ctx) error {
		path := c.Path()
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "fail",
			"message": fmt.Sprintf("Path: %v does not exists", path),
		})
	})
}
