# Telegram Data Research ServiceRU

Сервис, записывающий комментарии выбранных *Telegram* каналов.

Для начала необходимо клонировать гитхаб проект по адресу https://github.com/andreylikhterman/TelegramDataResearch/tree/with_subscription (именно ветка with_subscription). После чего нужно создать файл ```.env``` следующего вида:

```
NUM_OF_ACCOUNTS=X
TELEGRAM_API_ID=Y
TELEGRAM_API_HASH=Z
```
где ```NUM_OF_ACCOUNTS``` - количество *Telegram* аккаунтов, с которых будут просматриваться комментарии (через подписку на чат обсуждения, ограничение на аккаунт без *Premium* - 500), TELEGRAM_API_ID и TELEGRAM_API_HASH - поля, которые можно получить на оффициальном сайте https://my.telegram.org, зарегестрировав там аккаунт (одной регистрации хватит для всего сервиса).

Далее необходимо перейти по пути ```TelegramDataResearch/internal/application/telegram_data_research.go``` и там найти поле ```channels``` в функции ```Run``` (88 строка) и заполнить его строками с никами каналов (ВАЖНО: ники - не названия, ники - то, что стоит в ссылке на канал после **t.me/** - пример: название канала - Technodeus, ссылка на канал - t.me/technodeus2023, ник - technodeus2023). Для примера слайс заполнен следующим образом: ```"technodeus2023", "cherevatstreams"```

После необходимо запустить программу, после чего вам будет предложено написать номер телефона и код, пришедший по нему в Телеграм. Это будет предложено сделать столько раз, сколько аккаунтов указано в поле ```NUM_OF_ACCOUNTS```. Затем запустится приложение, все логи (включая комментарии) будут писаться в созданный файл ```app.log``` при количестве комментариев больше 10 (каждый 11ый будет запускать запись прошлых 10).

# Telegram Data Research ServiceENG

Service that records comments from selected *Telegram* channels.

To start, you need to clone the GitHub project from https://github.com/andreylikhterman/TelegramDataResearch/tree/with_subscription (specifically the **with_subscription** branch). Then, create a file named ```.env``` with the following content:

```
NUM_OF_ACCOUNTS=X
TELEGRAM_API_ID=Y
TELEGRAM_API_HASH=Z
```

where ```NUM_OF_ACCOUNTS``` is the number of *Telegram* accounts from which comments will be viewed (via subscription to the discussion chat; the limit for a non-*Premium* account is 500), and TELEGRAM_API_ID and TELEGRAM_API_HASH are the fields you can obtain on the official website https://my.telegram.org by registering an account there (one registration is enough for the entire service).

Next, navigate to ```TelegramDataResearch/internal/application/telegram_data_research.go``` and find the field ```channels``` in the ```Run``` function (line 88) and fill it with strings containing the channel usernames (IMPORTANT: usernames are not the channel names; usernames are what comes in the link after **t.me/** - for example: if the channel name is Technodeus, the channel link is t.me/technodeus2023, then the username is technodeus2023). In the example, the slice is filled as follows: ```"technodeus2023", "cherevatstreams"```.

After that, you need to run the program. You will then be prompted to enter a phone number and the code sent to it via Telegram. This will be prompted as many times as the number of accounts specified in the ```NUM_OF_ACCOUNTS``` field. Then, the application will start, and all logs (including comments) will be written to the created file ```app.log``` once the number of comments exceeds 10 (every 11th comment will trigger the recording of the previous 10).