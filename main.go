package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"go.uber.org/zap"
)

type Category struct {
	Slug       string
	Name       string
	Href       string
	ParentSlug string
}

type Recept struct {
	ID             int    `db:"id"`
	Name           string `db:"name"`
	CookingTime    string `db:"cooking_time"`
	Description    string `db:"description"`
	NumberServings string `db:"number_servings"`
	ImageSrc       string `db:"image_src"`
	Image          string `db:"image"`
	Href           string
	CategorySlug   string
}

func main() {
	// инициализируем логер для девелопмент среды
	logger, _ := zap.NewDevelopment()
	// запускаем парсинг категорий и присваиваем переменной categoryList результат в виде массива категорий
	categoryList := GetCategoryList(logger)

	// перебираем все категории
	for _, category := range categoryList {
		// получаем список рецептов категории
		recepties, err := GetRecepeList(category.Href, category.Slug, logger)

		// при возникновении ошибки, выводим в консоль
		if err != nil {
			logger.Error(err.Error())
		}

		// перебираем список рецептов
		for _, v := range recepties {
			// проходим по каждому рецепту и заходя на страницу парсим данные рецепта
			if err := GetRecepe(v, logger); err != nil {
				logger.Error(err.Error())
			}
		}

		// выводим полученные рецепты
		for _, r := range recepties {
			fmt.Println(r)
		}

	}
}

func GetCategoryList(logger *zap.Logger) []*Category {
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

	return listCategory
}

func GetRecepeList(urlCategory string, slugCategory string, logger *zap.Logger) ([]*Recept, error) {
	logger.Info(
		"Scraper: Получение списка ссылок на рецепты...",
	)
	// объявляем необходимые переменные
	var (
		count    = 0
		allCount = 0
		first    = true
		baseURL  = "https://eda.ru"
		c        = initColly()
		list     = []*Recept{}
	)

	// Ищем все элементы с указанным классом
	c.OnHTML(".emotion-1jdotsv", func(h *colly.HTMLElement) {
		// если это первый элемент, считываем содержимое и выводим в консоль (количество рецептов в категории)
		if first {
			allCountString := h.Text
			listStr := []string{
				"Найдено ",
				"Найден ",
				"Найдены ",
				" рецепта",
				" рецептов",
				" рецепт",
			}
			for _, v := range listStr {
				allCountString = strings.ReplaceAll(allCountString, v, "")
			}
			allCount, _ = strconv.Atoi(allCountString)

			first = false
			logger.Info(
				"Scraper: Всего рецептов по категории",
				zap.String("Категория", slugCategory),
				zap.Int("Всего", allCount),
			)
		}
	})

	// Ищем все элементы с указанным классом
	c.OnHTML(".emotion-1eugp2w", func(h *colly.HTMLElement) {
		recipeLink := h.ChildAttrs("a", "href")

		if recipeLink == nil {
			return
		}
		// добавляем рецепт в массив
		list = append(list, &Recept{Href: recipeLink[0], CategorySlug: slugCategory})

		count++
	})

	c.Visit(baseURL + urlCategory)

	lastCount := 0
	// если общее количество рецептов больше количества которое уже обработано, начинаем перебирать страницы
	if count < allCount {
		logger.Debug("Scraper: На первой странице не все рецепты")

		for i := 2; lastCount != count; i++ {
			lastCount = count

			// добавляем к ссылке параметр страницы
			url := fmt.Sprintf("%s?page=%s", baseURL+urlCategory, strconv.Itoa(i))

			logger.Info(
				"Scraper: Получение рецептов...",
				zap.Int("Получено", count),
				zap.Int("Всего", allCount),
				zap.String("Категория", slugCategory),
			)
			logger.Debug("Scraper: Парсинг", zap.Int("страница", i))

			c.Visit(url)
		}
	}
	logger.Info(
		"Scraper: Получено рецептов",
		zap.Int("Получено", count),
		zap.Int("Всего", allCount),
		zap.String("Категория", slugCategory),
	)

	// возвращаем массив рецептов
	return list, nil
}

func GetRecepe(recept *Recept, logger *zap.Logger) error {
	// объявляем необходимые переменные
	var (
		ss      = strings.Split(recept.Href, "-")
		id, _   = strconv.Atoi(ss[len(ss)-1])
		baseURL = "https://eda.ru"
		c       = initColly()
	)
	recept.ID = id

	// парсим картинку рецепта
	c.OnHTML("span[itemprop=resultPhoto]", func(h *colly.HTMLElement) {
		recept.ImageSrc = h.Attr("content")

	})
	// парсим название, время приготовления и количество порций
	c.OnHTML(".emotion-19rdt1j", func(h *colly.HTMLElement) {
		recept.Name = h.ChildText("h1")
		recept.CookingTime = h.ChildText(".emotion-my9yfq")
		recept.NumberServings = h.ChildText("span[itemprop=recipeYield]")

	})

	// парсим описание
	c.OnHTML(".emotion-aiknw3", func(h *colly.HTMLElement) {
		recept.Description = h.Text

	})

	c.Visit(baseURL + recept.Href)

	return nil
}

func initColly() *colly.Collector {

	// инициализируем новый коллектор с задержкой в 3 секунды чтобы не перегрузить сайт парсингом и не вызвать особых подозрений
	c := colly.NewCollector(colly.AllowedDomains("eda.ru"))
	c.Limit(&colly.LimitRule{
		DomainGlob: "eda.ru",
		Delay:      3 * time.Second,
	})
	return c
}
