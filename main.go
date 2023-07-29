package main

import (
	"fmt"
	"strings"

	"github.com/gocolly/colly"
	"go.uber.org/zap"
)

type Category struct {
	Slug       string
	Name       string
	Href       string
	ParentSlug string
}

func main() {

	// создаем массив категорий в который будем складывать все что спарсили с сайта
	listCategory := []*Category{}

	// инициализируем новый коллектор из пакета colly https://github.com/gocolly/colly
	// с его помощью будем парсить сайт
	c := colly.NewCollector(
		colly.AllowedDomains("eda.ru"),
	)

	// Ищем все элементы с указанными классами
	c.OnHTML(".emotion-18mh8uc .emotion-w5dos9", func(e *colly.HTMLElement) {
		ParentSlug := ""
		// Внутри каждого элемента выбираем дочерний элемент с указанным классом  (родительские категории)
		e.ForEach(".emotion-w5dos9", func(_ int, h *colly.HTMLElement) {
			// получаем ссылку на категорию
			categoryLinks := h.ChildAttrs("a", "href")

			// если ссылки нет, выходим из шага цикла
			if categoryLinks == nil {
				return
			}

			href := categoryLinks[0]
			ParentSlug = strings.ReplaceAll(href, "/recepty/", "")
			// получаем имя категории
			name := h.ChildText("a h3")
			// количесво рецептов в категории
			number := h.ChildText("a h3 span")
			// Заполняем структуру категории
			category := Category{
				Slug: ParentSlug,
				// удаляем из имени категории количество рецептов
				Name: strings.ReplaceAll(name, number, ""),
				Href: href,
			}
			// добавляем категорию в массив
			listCategory = append(listCategory, &category)
		})

		// получаем все дочерние категории
		e.ForEach(".emotion-8asrz1", func(_ int, h *colly.HTMLElement) {
			categoryLinks := h.ChildAttrs("a", "href")
			if categoryLinks == nil {
				return
			}

			href := categoryLinks[0]
			slug := strings.ReplaceAll(href, "/recepty/", "")
			name := h.ChildText("a span")
			number := h.ChildText("a span span")

			category := Category{
				Slug:       slug,
				Name:       strings.ReplaceAll(name, number, ""),
				Href:       href,
				ParentSlug: ParentSlug,
			}

			listCategory = append(listCategory, &category)
		})

	})
	// заходим на сайт и получаем разметку
	c.Visit("https://eda.ru")

	// инициализируем логер для продакшн среды
	logger, _ := zap.NewProduction()
	// выводим общее количество категорий которые удалось найти на странице
	logger.Info("Категории получены", zap.Int("количество", len(listCategory)))

	// проходим по всему массиву категорий и выводим в консоль названия
	for _, c := range listCategory {
		// Если категория является родительской то выводим просто название
		// Если категория дочерняя, выводим перед ней "-" чтобы визуально разделить список
		if len(c.ParentSlug) == 0 {
			fmt.Println(c.Name)
		} else {
			fmt.Println(" - " + c.Name)
		}
	}
}
