# MTPanel

Self-hosted веб-панель для управления **TeleMT** (`Go + SvelteKit + SQLite + systemd`).

## Установка в одну строку

```bash
curl -fsSL https://raw.githubusercontent.com/NikitaKHS/mtpanel/main/install.sh | sudo bash
```

С явной настройкой портов:

```bash
curl -fsSL https://raw.githubusercontent.com/NikitaKHS/mtpanel/main/install.sh | sudo bash -s -- --port 8080 --proxy-port 443
```

## Что делает установщик

1. Проверяет ОС/архитектуру и наличие `systemd`.
2. Создает системные директории и пользователя `mtpanel`.
3. Скачивает бинарник панели (или собирает из исходников при отсутствии релиза).
4. Устанавливает фронтенд-ассеты.
5. Пишет конфиг панели в `/etc/mtpanel/config.json`.
6. Ставит и запускает `mtpanel.service`.

## Первый запуск

1. Откройте `http://<SERVER_IP>:8080`.
2. При первом входе откройте `/setup` и задайте пароль администратора.
3. После этого войдите через `/login`.
4. В разделе `Прокси` нажмите установку TeleMT.

## Проверка после установки

```bash
sudo systemctl status mtpanel --no-pager
sudo journalctl -u mtpanel -n 200 --no-pager

sudo systemctl status telemt.service --no-pager
sudo journalctl -u telemt.service -n 200 --no-pager
```

## Частые проблемы

### `address already in use` на `:8080`

Порт занят другим процессом. Освободите порт или смените `--port`.

### `HTTP 404` на скачивании старого MTProxy

В актуальной версии используется только **TeleMT**, старый источник MTProxy больше не нужен.

## Примечания

- Поддерживается Linux + systemd.
- Режим только `telemt.service` (telemt-only).
- Для production рекомендуется reverse proxy + TLS.
