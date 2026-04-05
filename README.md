# MTPanel

Self-hosted веб-панель для управления MTProxy (`Go + SvelteKit + SQLite + systemd`).

## Установка одной строкой

```bash
curl -fsSL https://raw.githubusercontent.com/NikitaKHS/mtpanel/main/install.sh | sudo bash
```

С явной настройкой портов:

```bash
curl -fsSL https://raw.githubusercontent.com/NikitaKHS/mtpanel/main/install.sh | sudo bash -s -- --port 8080 --mtproxy-port 443
```

## Что делает установщик

1. Проверяет ОС, архитектуру и наличие `systemd`.
2. Создаёт пользователя `mtpanel` и системные директории.
3. Ставит бинарник панели из GitHub Release (или собирает из исходников, если релиз недоступен).
4. Ставит фронтенд-ассеты из релиза `web-dist.tar.gz`.
5. Если ассеты не найдены в релизе, собирает фронтенд из исходников по тегу релиза.
6. Пишет конфиг в `/etc/mtpanel/config.json`.
7. Создаёт и запускает `mtpanel.service`.

## Первый запуск

1. Откройте `http://<SERVER_IP>:8080`.
2. Если видите ответ `428 Precondition Required` на login: это нормально, сначала откройте `/setup`.
3. Задайте пароль администратора (от 12 до 128 символов).
4. После этого вход через `/login` начнёт работать.

## Быстрая проверка

1. В панели откройте `Proxy` и установите MTProxy.
2. Убедитесь, что статус стал `running`.
3. В `Links` создайте ссылку и проверьте её копирование.
4. В `Logs` проверьте, что логи читаются.

## Полезные команды

```bash
sudo systemctl status mtpanel
sudo systemctl restart mtpanel
sudo journalctl -u mtpanel -n 200 --no-pager

sudo systemctl status mtproto-proxy.service
sudo journalctl -u mtproto-proxy.service -n 200 --no-pager
```

## Типовые проблемы

### `no such table: app_settings`

Это означает, что не применились миграции БД. Обновите панель до последней версии и переустановите:

```bash
curl -fsSL https://raw.githubusercontent.com/NikitaKHS/mtpanel/main/install.sh | sudo bash
sudo systemctl restart mtpanel
```

### UI без стилей (чёрный текст на белом фоне)

Обычно это значит, что не установились статические фронтенд-файлы. Выполните переустановку командой выше и затем сделайте hard refresh в браузере (`Ctrl+F5`).

## Примечания

- Поддерживается Linux + systemd.
- Для production рекомендуется reverse proxy + TLS.
- Установщик идемпотентный, его можно запускать повторно для обновлений.
